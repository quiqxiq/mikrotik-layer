// ==================== services/mikrotik_service.go (WITH TIMEOUT FIX) ====================
package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"Mikrotik-Layer/models"
	"Mikrotik-Layer/repository"

	"github.com/go-routeros/routeros/v3"
)

// MikrotikConnection - Single router connection
type MikrotikConnection struct {
	RouterID   int
	Router     *models.Router
	Client     *routeros.Client
	mu         sync.RWMutex
	LastPing   time.Time
	IsHealthy  bool
}

// MikrotikService - Manages multiple router connections
type MikrotikService struct {
	connections map[int]*MikrotikConnection // RouterID -> Connection
	repo        *repository.RouterRepository
	mu          sync.RWMutex
}

// TrafficStats untuk menyimpan statistik traffic
type TrafficStats struct {
	RouterID      int
	InterfaceName string
	RxBytes       string
	TxBytes       string
	RxPackets     string
	TxPackets     string
	RxBitsPerSec  string
	TxBitsPerSec  string
	Timestamp     time.Time
}

var (
	serviceInstance *MikrotikService
	serviceOnce     sync.Once
)

// GetMikrotikService - Initialize service dengan repository
func GetMikrotikService(repo *repository.RouterRepository) *MikrotikService {
	serviceOnce.Do(func() {
		serviceInstance = &MikrotikService{
			connections: make(map[int]*MikrotikConnection),
			repo:        repo,
		}

		// Auto-connect ke semua active routers
		go serviceInstance.autoConnectActiveRouters()
		
		// Health check routine
		go serviceInstance.healthCheckRoutine()
	})

	return serviceInstance
}

// autoConnectActiveRouters - Connect ke semua router yang aktif
func (ms *MikrotikService) autoConnectActiveRouters() {
	routers, err := ms.repo.GetActiveRouters()
	if err != nil {
		log.Printf("Error loading active routers: %v", err)
		return
	}

	for _, router := range routers {
		if err := ms.ConnectRouter(router.ID); err != nil {
			log.Printf("Error auto-connecting to router %s (%d): %v", router.Name, router.ID, err)
		} else {
			log.Printf("✓ Auto-connected to router: %s (%s)", router.Name, router.Hostname)
		}
	}
}

// dialWithTimeout - Dial dengan timeout menggunakan context
func dialWithTimeout(address, username, password string, timeout time.Duration) (*routeros.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel untuk hasil
	type result struct {
		client *routeros.Client
		err    error
	}
	resultChan := make(chan result, 1)

	// Dial di goroutine
	go func() {
		// Create custom dialer dengan timeout
		dialer := &net.Dialer{
			Timeout: timeout,
		}
		
		// Dial TCP connection dulu
		conn, err := dialer.Dial("tcp", address)
		if err != nil {
			resultChan <- result{nil, fmt.Errorf("tcp dial failed: %w", err)}
			return
		}

		// Kemudian buat RouterOS client dari connection
		client, err := routeros.NewClient(conn)
		if err != nil {
			conn.Close()
			resultChan <- result{nil, fmt.Errorf("routeros client creation failed: %w", err)}
			return
		}

		// Login
		if err := client.Login(username, password); err != nil {
			client.Close()
			resultChan <- result{nil, fmt.Errorf("login failed: %w", err)}
			return
		}

		resultChan <- result{client, nil}
	}()

	// Wait dengan timeout
	select {
	case res := <-resultChan:
		return res.client, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("connection timeout after %v", timeout)
	}
}

// ConnectRouter - Connect ke router berdasarkan ID dari database (WITH TIMEOUT)
func (ms *MikrotikService) ConnectRouter(routerID int) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	log.Printf("Connecting to router ID: %d...", routerID)

	// Check if already connected
	if conn, exists := ms.connections[routerID]; exists {
		if conn.IsHealthy {
			log.Printf("Router ID %d already connected and healthy", routerID)
			return nil
		}
		// Close unhealthy connection
		log.Printf("Closing unhealthy connection for router ID %d", routerID)
		conn.Client.Close()
		delete(ms.connections, routerID)
	}

	// Load router config from database
	router, err := ms.repo.GetByID(routerID)
	if err != nil {
		return fmt.Errorf("router not found: %v", err)
	}

	log.Printf("Router config: %v", router)

	if !router.IsActive {
		return fmt.Errorf("router is not active")
	}

	// Create connection WITH TIMEOUT
	address := fmt.Sprintf("%s:%d", router.Hostname, router.Port)
	log.Printf("Dialing %s (timeout: 10s)...", address)
	
	client, err := dialWithTimeout(address, router.Username, router.Password, 20*time.Second)
	if err != nil {
		log.Printf("Failed to connect to router %s: %v", router.Name, err)
		// Update status to error
		ms.repo.UpdateStatus(routerID, &models.RouterStatusUpdate{
			Status: "error",
		})
		return fmt.Errorf("failed to connect: %v", err)
	}

	log.Printf("Connected to %s, getting system info...", router.Name)

	// Get system info
	systemInfo, _ := ms.getSystemInfo(client)
	
	// Update router status to online
	statusUpdate := &models.RouterStatusUpdate{
		Status: "online",
	}
	if systemInfo != nil {
		statusUpdate.Version = &systemInfo.Version
		statusUpdate.Uptime = &systemInfo.Uptime
	}
	ms.repo.UpdateStatus(routerID, statusUpdate)

	// Store connection
	ms.connections[routerID] = &MikrotikConnection{
		RouterID:  routerID,
		Router:    router,
		Client:    client,
		LastPing:  time.Now(),
		IsHealthy: true,
	}

	log.Printf("✓ Successfully connected to router: %s (%s)", router.Name, router.Hostname)
	return nil
}

// DisconnectRouter - Disconnect dari router
func (ms *MikrotikService) DisconnectRouter(routerID int) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	conn, exists := ms.connections[routerID]
	if !exists {
		return fmt.Errorf("router not connected")
	}

	conn.Client.Close()
	delete(ms.connections, routerID)

	// Update status to offline
	ms.repo.UpdateStatus(routerID, &models.RouterStatusUpdate{
		Status: "offline",
	})

	log.Printf("✓ Disconnected from router ID: %d", routerID)
	return nil
}

// GetConnection - Get connection untuk router tertentu
func (ms *MikrotikService) GetConnection(routerID int) (*MikrotikConnection, error) {
	ms.mu.RLock()
	conn, exists := ms.connections[routerID]
	ms.mu.RUnlock()

	if !exists {
		// Try to connect
		if err := ms.ConnectRouter(routerID); err != nil {
			return nil, fmt.Errorf("router not connected: %v", err)
		}
		ms.mu.RLock()
		conn = ms.connections[routerID]
		ms.mu.RUnlock()
	}

	if !conn.IsHealthy {
		return nil, fmt.Errorf("router connection unhealthy")
	}

	return conn, nil
}

// GetAllConnections - Get semua active connections
func (ms *MikrotikService) GetAllConnections() map[int]*MikrotikConnection {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Return copy
	result := make(map[int]*MikrotikConnection)
	for k, v := range ms.connections {
		result[k] = v
	}
	return result
}

// healthCheckRoutine - Periodic health check untuk semua connections
func (ms *MikrotikService) healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ms.mu.RLock()
		connections := make([]*MikrotikConnection, 0, len(ms.connections))
		for _, conn := range ms.connections {
			connections = append(connections, conn)
		}
		ms.mu.RUnlock()

		for _, conn := range connections {
			go ms.checkConnection(conn)
		}
	}
}

// checkConnection - Check single connection health
func (ms *MikrotikService) checkConnection(conn *MikrotikConnection) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// Try to ping
	_, err := conn.Client.RunArgs([]string{"/system/resource/print"})
	if err != nil {
		conn.IsHealthy = false
		log.Printf("✗ Router %s unhealthy: %v", conn.Router.Name, err)
		
		ms.repo.UpdateStatus(conn.RouterID, &models.RouterStatusUpdate{
			Status: "error",
		})
		
		// Try to reconnect
		go ms.ConnectRouter(conn.RouterID)
		return
	}

	conn.IsHealthy = true
	conn.LastPing = time.Now()

	// Get system info and update status
	systemInfo, _ := ms.getSystemInfo(conn.Client)
	statusUpdate := &models.RouterStatusUpdate{
		Status: "online",
	}
	if systemInfo != nil {
		statusUpdate.Version = &systemInfo.Version
		statusUpdate.Uptime = &systemInfo.Uptime
	}
	ms.repo.UpdateStatus(conn.RouterID, statusUpdate)
}

// SystemInfo struct
type SystemInfo struct {
	Version string
	Uptime  string
}

// getSystemInfo - Get system resource info
func (ms *MikrotikService) getSystemInfo(client *routeros.Client) (*SystemInfo, error) {
	r, err := client.RunArgs([]string{"/system/resource/print"})
	if err != nil {
		return nil, err
	}

	if len(r.Re) == 0 {
		return nil, fmt.Errorf("no system info")
	}

	return &SystemInfo{
		Version: r.Re[0].Map["version"],
		Uptime:  r.Re[0].Map["uptime"],
	}, nil
}

// ==================== Interface Methods ====================

func (ms *MikrotikService) GetInterfaces(routerID int) ([]*models.Interface, error) {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return nil, err
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	r, err := conn.Client.Run(
		"/interface/print",
		"=.proplist=.id,name,type,running,disabled,rx-bytes,tx-bytes,rx-packets,tx-packets",
	)
	if err != nil {
		return nil, err
	}

	var interfaces []*models.Interface
	for _, re := range r.Re {
		iface := &models.Interface{
			Name:      re.Map["name"],
			Type:      re.Map["type"],
			Running:   re.Map["running"] == "true",
			Disabled:  re.Map["disabled"] == "true",
			RxBytes:   re.Map["rx-bytes"],
			TxBytes:   re.Map["tx-bytes"],
			RxPackets: re.Map["rx-packets"],
			TxPackets: re.Map["tx-packets"],
		}
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

func (ms *MikrotikService) EnableInterface(routerID int, name string) error {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	r, err := conn.Client.Run("/interface/print", fmt.Sprintf("?name=%s", name))
	if err != nil {
		return err
	}

	if len(r.Re) == 0 {
		return fmt.Errorf("interface %s not found", name)
	}

	id := r.Re[0].Map[".id"]
	_, err = conn.Client.Run("/interface/set",
		fmt.Sprintf("=.id=%s", id),
		"=disabled=false")

	return err
}

func (ms *MikrotikService) DisableInterface(routerID int, name string) error {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	r, err := conn.Client.Run("/interface/print", fmt.Sprintf("?name=%s", name))
	if err != nil {
		return err
	}

	if len(r.Re) == 0 {
		return fmt.Errorf("interface %s not found", name)
	}

	id := r.Re[0].Map[".id"]
	_, err = conn.Client.Run("/interface/set",
		fmt.Sprintf("=.id=%s", id),
		"=disabled=true")

	return err
}

// ==================== Address Methods ====================

func (ms *MikrotikService) GetAddresses(routerID int) ([]*models.Address, error) {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return nil, err
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	r, err := conn.Client.Run(
		"/ip/address/print",
		"=.proplist=.id,address,interface,network,disabled",
	)
	if err != nil {
		return nil, err
	}

	var addresses []*models.Address
	for _, re := range r.Re {
		addr := &models.Address{
			ID:        re.Map[".id"],
			Address:   re.Map["address"],
			Interface: re.Map["interface"],
			Network:   re.Map["network"],
			Disabled:  re.Map["disabled"] == "true",
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

func (ms *MikrotikService) AddAddress(routerID int, iface, address string) error {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	_, err = conn.Client.Run("/ip/address/add",
		fmt.Sprintf("=address=%s", address),
		fmt.Sprintf("=interface=%s", iface))

	return err
}

func (ms *MikrotikService) RemoveAddress(routerID int, id string) error {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	_, err = conn.Client.Run("/ip/address/remove",
		fmt.Sprintf("=.id=%s", id))

	return err
}

// ==================== Queue Methods ====================

func (ms *MikrotikService) GetQueues(routerID int) ([]*models.Queue, error) {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return nil, err
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	r, err := conn.Client.Run(
		"/queue/simple/print",
		"=.proplist=.id,name,target,max-limit,burst-limit,disabled",
	)
	if err != nil {
		return nil, err
	}

	var queues []*models.Queue
	for _, re := range r.Re {
		queue := &models.Queue{
			ID:         re.Map[".id"],
			Name:       re.Map["name"],
			Target:     re.Map["target"],
			MaxLimit:   re.Map["max-limit"],
			BurstLimit: re.Map["burst-limit"],
			Disabled:   re.Map["disabled"] == "true",
		}
		queues = append(queues, queue)
	}

	return queues, nil
}

func (ms *MikrotikService) AddQueue(routerID int, name, target, maxLimit string) error {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	_, err = conn.Client.Run("/queue/simple/add",
		fmt.Sprintf("=name=%s", name),
		fmt.Sprintf("=target=%s", target),
		fmt.Sprintf("=max-limit=%s", maxLimit))

	return err
}

func (ms *MikrotikService) RemoveQueue(routerID int, id string) error {
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	_, err = conn.Client.Run("/queue/simple/remove",
		fmt.Sprintf("=.id=%s", id))

	return err
}

// ==================== Traffic Monitoring ====================

// ==================== FIXED MonitorInterfaceTraffic ====================
// Replace in mikrotik_service.go

func (ms *MikrotikService) MonitorInterfaceTraffic(routerID int, interfaceName string, callback func(TrafficStats)) error {
	log.Printf("[MONITOR] Starting monitor for router %d, interface %s", routerID, interfaceName)
	
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		log.Printf("[MONITOR] Failed to get connection: %v", err)
		return err
	}

	// ✅ JANGAN LOCK DI SINI - Listen() akan handle concurrent access
	log.Printf("[MONITOR] Calling RouterOS Listen command...")
	
	listen, err := conn.Client.Listen(
		"/interface/monitor-traffic",
		fmt.Sprintf("=interface=%s", interfaceName),
	)
	if err != nil {
		log.Printf("[MONITOR] Listen command failed: %v", err)
		return fmt.Errorf("failed to start monitoring: %v", err)
	}

	log.Printf("[MONITOR] Listen command successful, starting goroutine...")

	go func() {
		defer func() {
			log.Printf("[MONITOR] Goroutine stopping, canceling listener...")
			listen.Cancel()
		}()

		updateCount := 0
		log.Printf("[MONITOR] Waiting for data from RouterOS...")
		
		for {
			sentence, more := <-listen.Chan()
			if !more {
				log.Printf("[MONITOR] Channel closed for router %d, interface %s", routerID, interfaceName)
				return
			}

			updateCount++
			
			// Debug: Log first few sentences
			// if updateCount <= 5 {
			// 	log.Printf("[MONITOR] Update #%d - Received sentence: Word=%s", updateCount, sentence.Word)
			// 	if sentence.Word == "!re" {
			// 		log.Printf("[MONITOR]   Data: rx-bytes=%s, tx-bytes=%s, rx-bps=%s, tx-bps=%s",
			// 			sentence.Map["rx-bytes"],
			// 			sentence.Map["tx-bytes"],
			// 			sentence.Map["rx-bits-per-second"],
			// 			sentence.Map["tx-bits-per-second"])
			// 	}
			// }

			if sentence.Word == "!trap" {
				log.Printf("[MONITOR] RouterOS trap/error: %+v", sentence.Map)
				continue
			}

			if sentence.Word == "!done" {
				log.Printf("[MONITOR] RouterOS sent !done")
				continue
			}

			if sentence.Word != "!re" {
				if updateCount <= 5 {
					log.Printf("[MONITOR] Skipping sentence with word: %s", sentence.Word)
				}
				continue
			}

			stats := TrafficStats{
				RouterID:      routerID,
				InterfaceName: interfaceName,
				RxBytes:       sentence.Map["rx-bytes"],
				TxBytes:       sentence.Map["tx-bytes"],
				RxPackets:     sentence.Map["rx-packets"],
				TxPackets:     sentence.Map["tx-packets"],
				RxBitsPerSec:  sentence.Map["rx-bits-per-second"],
				TxBitsPerSec:  sentence.Map["tx-bits-per-second"],
				Timestamp:     time.Now(),
			}

			if updateCount <= 3 {
				log.Printf("[MONITOR] Calling callback with stats...")
			}

			callback(stats)

			if updateCount == 5 {
				log.Printf("[MONITOR] (Further detailed logs suppressed, monitoring continues...)")
			}
		}
	}()

	log.Printf("[MONITOR] Monitor setup complete for router %d, interface %s", routerID, interfaceName)
	return nil
}

// GetInterfaceTrafficOnce - Keep with lock since it's one-time operation
func (ms *MikrotikService) GetInterfaceTrafficOnce(routerID int, interfaceName string) (*TrafficStats, error) {
	log.Printf("[TRAFFIC-ONCE] Getting traffic for router %d, interface %s", routerID, interfaceName)
	
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		log.Printf("[TRAFFIC-ONCE] Failed to get connection: %v", err)
		return nil, err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	log.Printf("[TRAFFIC-ONCE] Executing monitor-traffic command...")
	r, err := conn.Client.RunArgs([]string{
		"/interface/monitor-traffic",
		fmt.Sprintf("=interface=%s", interfaceName),
		"=once=",
	})
	if err != nil {
		log.Printf("[TRAFFIC-ONCE] Command failed: %v", err)
		return nil, err
	}

	log.Printf("[TRAFFIC-ONCE] Command successful, got %d results", len(r.Re))

	if len(r.Re) == 0 {
		log.Printf("[TRAFFIC-ONCE] No data returned for interface %s", interfaceName)
		
		// Try to list available interfaces
		log.Printf("[TRAFFIC-ONCE] Attempting to list available interfaces...")
		ifaceResult, ifaceErr := conn.Client.Run("/interface/print", "=.proplist=name")
		if ifaceErr == nil && len(ifaceResult.Re) > 0 {
			var names []string
			for _, re := range ifaceResult.Re {
				names = append(names, re.Map["name"])
			}
			log.Printf("[TRAFFIC-ONCE] Available interfaces: %v", names)
		}
		
		return nil, fmt.Errorf("interface %s not found or no data", interfaceName)
	}

	re := r.Re[0]
	log.Printf("[TRAFFIC-ONCE] Response map keys: %v", func() []string {
		keys := make([]string, 0, len(re.Map))
		for k := range re.Map {
			keys = append(keys, k)
		}
		return keys
	}())

	stats := &TrafficStats{
		RouterID:      routerID,
		InterfaceName: interfaceName,
		RxBytes:       re.Map["rx-bytes"],
		TxBytes:       re.Map["tx-bytes"],
		RxPackets:     re.Map["rx-packets"],
		TxPackets:     re.Map["tx-packets"],
		RxBitsPerSec:  re.Map["rx-bits-per-second"],
		TxBitsPerSec:  re.Map["tx-bits-per-second"],
		Timestamp:     time.Now(),
	}

	log.Printf("[TRAFFIC-ONCE] Stats created: RX=%s bytes, TX=%s bytes, RX-Speed=%s bps", 
		stats.RxBytes, stats.TxBytes, stats.RxBitsPerSec)
	return stats, nil
}

// ==================== ADD TO mikrotik_service.go ====================
// Replace MonitorInterfaceTraffic with this version that supports context

func (ms *MikrotikService) MonitorInterfaceTrafficWithContext(ctx context.Context, routerID int, interfaceName string, callback func(TrafficStats)) error {
	log.Printf("[MONITOR] Starting monitor for router %d, interface %s", routerID, interfaceName)
	
	conn, err := ms.GetConnection(routerID)
	if err != nil {
		log.Printf("[MONITOR] Failed to get connection: %v", err)
		return err
	}

	log.Printf("[MONITOR] Calling RouterOS Listen command...")
	
	listen, err := conn.Client.Listen(
		"/interface/monitor-traffic",
		fmt.Sprintf("=interface=%s", interfaceName),
	)
	if err != nil {
		log.Printf("[MONITOR] Listen command failed: %v", err)
		return fmt.Errorf("failed to start monitoring: %v", err)
	}

	log.Printf("[MONITOR] Listen command successful, starting goroutine...")

	go func() {
		defer func() {
			log.Printf("[MONITOR] Canceling listener for router %d, interface %s", routerID, interfaceName)
			listen.Cancel()
		}()

		updateCount := 0
		log.Printf("[MONITOR] Waiting for data from RouterOS...")
		
		for {
			select {
			case <-ctx.Done():
				log.Printf("[MONITOR] Context canceled for router %d, interface %s - stopping monitoring", routerID, interfaceName)
				return
				
			case sentence, more := <-listen.Chan():
				if !more {
					log.Printf("[MONITOR] Channel closed for router %d, interface %s", routerID, interfaceName)
					return
				}

				updateCount++
				
				// Debug: Log first few sentences
				// if updateCount <= 5 {
				// 	log.Printf("[MONITOR] Update #%d - Received sentence: Word=%s", updateCount, sentence.Word)
				// 	if sentence.Word == "!re" {
				// 		log.Printf("[MONITOR]   Data: rx-bytes=%s, tx-bytes=%s, rx-bps=%s, tx-bps=%s",
				// 			sentence.Map["rx-bytes"],
				// 			sentence.Map["tx-bytes"],
				// 			sentence.Map["rx-bits-per-second"],
				// 			sentence.Map["tx-bits-per-second"])
				// 	}
				// }

				if sentence.Word == "!trap" {
					log.Printf("[MONITOR] RouterOS trap/error: %+v", sentence.Map)
					continue
				}

				if sentence.Word == "!done" {
					log.Printf("[MONITOR] RouterOS sent !done")
					continue
				}

				if sentence.Word != "!re" {
					if updateCount <= 5 {
						log.Printf("[MONITOR] Skipping sentence with word: %s", sentence.Word)
					}
					continue
				}

				stats := TrafficStats{
					RouterID:      routerID,
					InterfaceName: interfaceName,
					RxBytes:       sentence.Map["rx-bytes"],
					TxBytes:       sentence.Map["tx-bytes"],
					RxPackets:     sentence.Map["rx-packets"],
					TxPackets:     sentence.Map["tx-packets"],
					RxBitsPerSec:  sentence.Map["rx-bits-per-second"],
					TxBitsPerSec:  sentence.Map["tx-bits-per-second"],
					Timestamp:     time.Now(),
				}

				if updateCount <= 3 {
					log.Printf("[MONITOR] Calling callback with stats...")
				}

				// Check context before calling callback
				select {
				case <-ctx.Done():
					log.Printf("[MONITOR] Context canceled before callback")
					return
				default:
					callback(stats)
				}

				if updateCount == 5 {
					log.Printf("[MONITOR] (Further detailed logs suppressed, monitoring continues...)")
				}
			}
		}
	}()

	log.Printf("[MONITOR] Monitor setup complete for router %d, interface %s", routerID, interfaceName)
	return nil
}

// Keep the old method for backward compatibility


// ==================== IMPORTANT NOTE ====================
// The Listen() method from go-routeros is designed to handle concurrent access
// internally. Adding external locks can actually cause deadlocks or prevent
// the background goroutine from receiving data properly.
// 
// Only use locks for Run() or RunArgs() which are synchronous operations.

func (ms *MikrotikService) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for routerID, conn := range ms.connections {
		if err := conn.Client.Close(); err != nil {
			log.Printf("Error closing connection to router %d: %v", routerID, err)
		}
	}

	ms.connections = make(map[int]*MikrotikConnection)
	return nil
}