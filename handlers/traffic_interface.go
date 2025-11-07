package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"Mikrotik-Layer/models"
	"Mikrotik-Layer/services"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type TrafficMessage struct {
	Type      string                 `json:"type"`
	Interface string                 `json:"interface,omitempty"`
	Data      *services.TrafficStats `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// MonitorTrafficWS - WebSocket untuk monitoring traffic multiple interfaces (same router)
// Patterns:
// - Single interface: /ws/traffic/monitor?router_id=1&interface=ether1
// - Multiple interfaces: /ws/traffic/monitor?router_id=1&interfaces=ether1,ether2,ether3
func MonitorTrafficWS(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[WS] New connection attempt from %s", r.RemoteAddr)
		
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WS] Error upgrade WebSocket: %v", err)
			return
		}
		defer conn.Close()

		// Parse router_id
		routerID, err := strconv.Atoi(r.URL.Query().Get("router_id"))
		if err != nil || routerID == 0 {
			log.Printf("[WS] Invalid router_id parameter")
			sendMessage(conn, TrafficMessage{
				Type:      "error",
				Error:     "parameter 'router_id' diperlukan dan harus valid",
				Timestamp: time.Now(),
			})
			return
		}

		// Parse interfaces
		interfaces := parseInterfaceList(r)
		if len(interfaces) == 0 {
			log.Printf("[WS] No interfaces specified")
			sendMessage(conn, TrafficMessage{
				Type:      "error",
				Error:     "parameter 'interface' atau 'interfaces' diperlukan",
				Timestamp: time.Now(),
			})
			return
		}

		log.Printf("[WS] Connection established - Router ID: %d, Interfaces: %v", routerID, interfaces)

		// Context untuk cancel semua monitoring
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Channels untuk koordinasi
		done := make(chan bool, 1)
		
		// Mutex untuk protect WebSocket writes
		var wsMutex sync.Mutex
		wsOpen := true

		// Counter untuk setiap interface
		updateCounters := make(map[string]int)
		var counterMutex sync.Mutex

		// Goroutine untuk baca message dari client (keep-alive & detect disconnect)
		go func() {
			defer func() {
				log.Printf("[WS] Read goroutine stopping for router %d", routerID)
				cancel() // Cancel all monitoring when client disconnects
				done <- true
			}()
			for {
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("[WS] Client disconnected (router %d): %v", routerID, err)
					return
				}
				
				// Handle ping/pong or commands
				if messageType == websocket.TextMessage {
					var cmd map[string]interface{}
					if err := json.Unmarshal(message, &cmd); err == nil {
						if cmdType, ok := cmd["type"].(string); ok && cmdType == "ping" {
							wsMutex.Lock()
							if wsOpen {
								sendMessage(conn, TrafficMessage{
									Type:      "pong",
									Timestamp: time.Now(),
								})
							}
							wsMutex.Unlock()
						}
					}
				}
			}
		}()

		// Start monitoring untuk setiap interface
		var wg sync.WaitGroup
		startErrors := make([]string, 0)
		var startErrorMutex sync.Mutex

		for _, iface := range interfaces {
			wg.Add(1)
			go func(interfaceName string) {
				defer wg.Done()

				log.Printf("[WS] Starting monitor for router %d, interface %s", routerID, interfaceName)
				
				// Callback untuk traffic updates
				callback := func(stats services.TrafficStats) {
					select {
					case <-ctx.Done():
						return
					default:
					}

					// Update counter
					counterMutex.Lock()
					updateCounters[interfaceName]++
					// count := updateCounters[interfaceName]
					counterMutex.Unlock()
					

					msg := TrafficMessage{
						Type:      "traffic_update",
						Interface: interfaceName,
						Data:      &stats,
						Timestamp: time.Now(),
					}

					// Safe write dengan mutex
					wsMutex.Lock()
					if wsOpen {
						if err := conn.WriteJSON(msg); err != nil {
							log.Printf("[WS] Error sending data (%s): %v", interfaceName, err)
							wsOpen = false
							cancel()
						}
					}
					wsMutex.Unlock()
				}

				// Start monitoring dengan context
				if err := ms.MonitorInterfaceTrafficWithContext(ctx, routerID, interfaceName, callback); err != nil {
					log.Printf("[WS] Failed to start monitoring interface %s: %v", interfaceName, err)
					
					startErrorMutex.Lock()
					startErrors = append(startErrors, fmt.Sprintf("%s: %v", interfaceName, err))
					startErrorMutex.Unlock()
				}
			}(iface)
		}

		// Wait sebentar untuk memastikan semua monitoring dimulai
		time.Sleep(500 * time.Millisecond)

		// Send status message
		wsMutex.Lock()
		if len(startErrors) > 0 {
			errMsg := fmt.Sprintf("Failed to start %d interface(s): %s", 
				len(startErrors), strings.Join(startErrors, "; "))
			log.Printf("[WS] %s", errMsg)
			
			if wsOpen {
				sendMessage(conn, TrafficMessage{
					Type:      "error",
					Error:     errMsg,
					Timestamp: time.Now(),
				})
			}
			
			// Jika semua gagal, return
			if len(startErrors) == len(interfaces) {
				wsMutex.Unlock()
				return
			}
		}

		// Send success message untuk yang berhasil
		successCount := len(interfaces) - len(startErrors)
		if successCount > 0 && wsOpen {
			successMsg := TrafficMessage{
				Type:      "connected",
				Message:   fmt.Sprintf("Monitoring started for router %d: %s (%d interface(s))", 
					routerID, strings.Join(interfaces, ", "), successCount),
				Timestamp: time.Now(),
			}
			sendMessage(conn, successMsg)
			log.Printf("[WS] Success message sent to client")
		}
		wsMutex.Unlock()

		// Wait until done
		<-done
		
		// Mark WebSocket as closed
		wsMutex.Lock()
		wsOpen = false
		wsMutex.Unlock()
		
		// Log final statistics
		counterMutex.Lock()
		totalUpdates := 0
		for iface, count := range updateCounters {
			log.Printf("[WS] Interface %s: %d updates", iface, count)
			totalUpdates += count
		}
		counterMutex.Unlock()
		
		log.Printf("[WS] Monitoring stopped - Router %d, Total updates: %d", routerID, totalUpdates)
	}
}

// parseInterfaceList parses interface parameter(s) from URL
func parseInterfaceList(r *http.Request) []string {
	query := r.URL.Query()
	var interfaces []string

	// Try "interfaces" parameter (comma-separated list)
	if interfacesParam := query.Get("interfaces"); interfacesParam != "" {
		parts := strings.Split(interfacesParam, ",")
		for _, iface := range parts {
			if iface = strings.TrimSpace(iface); iface != "" {
				interfaces = append(interfaces, iface)
			}
		}
		return interfaces
	}

	// Fallback to single "interface" parameter (backward compatible)
	if interfaceName := query.Get("interface"); interfaceName != "" {
		interfaces = append(interfaces, strings.TrimSpace(interfaceName))
		return interfaces
	}

	return interfaces
}

// sendMessage is a helper to safely send messages
func sendMessage(conn *websocket.Conn, msg TrafficMessage) {
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("[WS] Error sending message: %v", err)
	}
}

// GetTrafficOnce - HTTP endpoint untuk get traffic stats
func GetTrafficOnce(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] GetTrafficOnce request from %s", r.RemoteAddr)
		
		routerID, err := strconv.Atoi(r.URL.Query().Get("router_id"))
		if err != nil || routerID == 0 {
			log.Printf("[HTTP] Invalid router_id parameter")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'router_id' diperlukan",
			})
			return
		}

		interfaceName := r.URL.Query().Get("interface")
		if interfaceName == "" {
			log.Printf("[HTTP] Missing interface parameter")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'interface' diperlukan",
			})
			return
		}

		log.Printf("[HTTP] Getting traffic stats for router %d, interface %s", routerID, interfaceName)

		stats, err := ms.GetInterfaceTrafficOnce(routerID, interfaceName)
		if err != nil {
			log.Printf("[HTTP] Error getting traffic stats: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		log.Printf("[HTTP] Traffic stats retrieved successfully: RX=%s, TX=%s", 
			stats.RxBytes, stats.TxBytes)

		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Data:    stats,
		})
	}
}

// ListAvailableInterfaces - Get list of available interfaces for monitoring
func ListAvailableInterfaces(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] ListAvailableInterfaces request")
		
		routerID, err := strconv.Atoi(r.URL.Query().Get("router_id"))
		if err != nil || routerID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'router_id' diperlukan dan harus valid",
			})
			return
		}

		interfaces, err := ms.GetInterfaces(routerID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// Filter only running interfaces
		var available []map[string]interface{}
		for _, iface := range interfaces {
			if iface.Running && !iface.Disabled {
				available = append(available, map[string]interface{}{
					"name":       iface.Name,
					"type":       iface.Type,
					"rx_bytes":   iface.RxBytes,
					"tx_bytes":   iface.TxBytes,
					"rx_packets": iface.RxPackets,
					"tx_packets": iface.TxPackets,
				})
			}
		}

		log.Printf("[HTTP] Found %d available interfaces for router %d", len(available), routerID)

		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Data:    available,
			Message: fmt.Sprintf("Found %d available interfaces", len(available)),
		})
	}
}

// GetConnectionStatus - Get status semua router connections
func GetConnectionStatus(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] GetConnectionStatus request")
		
		connections := ms.GetAllConnections()

		type ConnectionInfo struct {
			RouterID   int       `json:"router_id"`
			RouterName string    `json:"router_name"`
			Hostname   string    `json:"hostname"`
			IsHealthy  bool      `json:"is_healthy"`
			LastPing   time.Time `json:"last_ping"`
		}

		var result []ConnectionInfo
		for _, conn := range connections {
			result = append(result, ConnectionInfo{
				RouterID:   conn.RouterID,
				RouterName: conn.Router.Name,
				Hostname:   conn.Router.Hostname,
				IsHealthy:  conn.IsHealthy,
				LastPing:   conn.LastPing,
			})
		}

		log.Printf("[HTTP] Found %d active connections", len(result))
		
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Data:    result,
		})
	}
}

// ConnectRouterHandler - Manual connect ke router dengan timeout
func ConnectRouterHandler(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerID, err := strconv.Atoi(r.URL.Query().Get("router_id"))
		if err != nil || routerID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'router_id' diperlukan",
			})
			return
		}

		log.Printf("[HTTP] Attempting to connect to router ID: %d", routerID)

		// Gunakan context dengan timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Channel untuk hasil connection
		resultChan := make(chan error, 1)

		// Jalankan connection di goroutine
		go func() {
			resultChan <- ms.ConnectRouter(routerID)
		}()

		// Wait dengan timeout
		select {
		case err := <-resultChan:
			if err != nil {
				log.Printf("[HTTP] Failed to connect router ID %d: %v", routerID, err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.ApiResponse{
					Success: false,
					Error:   err.Error(),
				})
				return
			}

			log.Printf("[HTTP] Successfully connected to router ID: %d", routerID)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: true,
				Message: "Router berhasil terkoneksi",
			})

		case <-ctx.Done():
			log.Printf("[HTTP] Connection timeout for router ID: %d", routerID)
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "Connection timeout after 30 seconds",
			})
		}
	}
}

// DisconnectRouterHandler - Manual disconnect dari router
func DisconnectRouterHandler(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerID, err := strconv.Atoi(r.URL.Query().Get("router_id"))
		if err != nil || routerID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'router_id' diperlukan",
			})
			return
		}

		log.Printf("[HTTP] Disconnecting router ID: %d", routerID)

		if err := ms.DisconnectRouter(routerID); err != nil {
			log.Printf("[HTTP] Failed to disconnect router ID %d: %v", routerID, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		log.Printf("[HTTP] Successfully disconnected router ID: %d", routerID)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Message: "Router berhasil didisconnect",
		})
	}
}

// HealthCheck - Simple health check endpoint
func WsHealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(models.ApiResponse{
		Success: true,
		Message: "WebSocket server is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
			"status":    "ok",
		},
	})
}