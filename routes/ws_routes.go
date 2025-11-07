// ==================== routes/websocket_routes.go ====================
package routes

import (
	"log"
	"net/http"
	"time"

	"Mikrotik-Layer/database"
	"Mikrotik-Layer/handlers"
	"Mikrotik-Layer/middleware"
	"Mikrotik-Layer/repository"
	"Mikrotik-Layer/services"
)

func SetupWebSocketRoutes(db *database.Database) *http.ServeMux {
	routerRepo := repository.NewRouterRepository(db.DB)
	ms := services.GetMikrotikService(routerRepo)

	mux := http.NewServeMux()

	// ==================== WebSocket Endpoints ====================
	
	// Real-time interface traffic monitoring
	// Single interface: ?router_id=1&interface=ether1
	// Multiple interfaces: ?router_id=1&interfaces=ether1,ether2,ether3
	mux.HandleFunc("/ws/traffic/monitor", handlers.MonitorTrafficWS(ms))

	// ==================== HTTP API Endpoints ====================
	
	// Get single interface traffic stats
	mux.HandleFunc("/api/traffic/once", middleware.JSONMiddleware(handlers.GetTrafficOnce(ms)))
	
	// List available interfaces for monitoring
	mux.HandleFunc("/api/interfaces/list", middleware.JSONMiddleware(handlers.ListAvailableInterfaces(ms)))

	// Health check endpoint
	mux.HandleFunc("/ws/health", middleware.JSONMiddleware(handlers.HealthCheck))

	// ==================== Connection Management ====================
	
	mux.HandleFunc("/api/ws/connections/status", middleware.JSONMiddleware(handlers.GetConnectionStatus(ms)))
	mux.HandleFunc("/api/ws/connections/connect", middleware.JSONMiddleware(handlers.ConnectRouterHandler(ms)))
	mux.HandleFunc("/api/ws/connections/disconnect", middleware.JSONMiddleware(handlers.DisconnectRouterHandler(ms)))

	log.Println("✓ WebSocket routes configured successfully")
	log.Println("  ┌─ WebSocket Endpoint:")
	log.Println("  │  • /ws/traffic/monitor")
	log.Println("  │    - Single: ?router_id=1&interface=ether1")
	log.Println("  │    - Multi:  ?router_id=1&interfaces=ether1,ether2,ether3")
	log.Println("  │")
	log.Println("  ├─ HTTP API Endpoints:")
	log.Println("  │  • /api/traffic/once?router_id=X&interface=Y")
	log.Println("  │  • /api/interfaces/list?router_id=X")
	log.Println("  │")
	log.Println("  └─ Management:")
	log.Println("     • /ws/health")
	log.Println("     • /api/ws/connections/status")

	return mux
}

// SetupWebSocketServer untuk setup server dengan custom config
func SetupWebSocketServer(db *database.Database, addr string) *http.Server {
	mux := SetupWebSocketRoutes(db)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,  // Increased for WebSocket
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second, // Increased for long-lived connections
	}

	return server
}