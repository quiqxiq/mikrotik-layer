package main

import (
	"log"
	"net/http"

	"Mikrotik-Layer/config"
	"Mikrotik-Layer/database"
	"Mikrotik-Layer/routes"
)

func main() {
	log.Println("🚀 Starting Mikrotik Layer API...")

	// Load configuration
	cfg := config.LoadConfig()
	log.Println("✓ Configuration loaded")

	// Initialize database
	db, err := database.NewDatabase(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	defer db.Close()
	log.Println("✓ Database connected")

	// Setup REST API router (port 8080)
	restRouter := routes.SetupRoutes(db)

	// Setup WebSocket router (port 8081)
	wsRouter := routes.SetupWebSocketRoutes(db)

	// Run REST API server
	go func() {
		log.Printf("🌐 REST API Server listening on %s\n", cfg.ServerAddr)
		if err := http.ListenAndServe(cfg.ServerAddr, restRouter); err != nil {
			log.Fatal("❌ REST API server error:", err)
		}
	}()

	// Run WebSocket server
	log.Printf("🔌 WebSocket Server listening on %s\n", cfg.WSServerAddr)
	if err := http.ListenAndServe(cfg.WSServerAddr, wsRouter); err != nil {
		log.Fatal("❌ WebSocket server error:", err)
	}
}