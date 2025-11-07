package routes

import (
	"log"
	"net/http"
	"strings"

	"Mikrotik-Layer/database"
	"Mikrotik-Layer/handlers"
	"Mikrotik-Layer/middleware"
	"Mikrotik-Layer/repository"
	"Mikrotik-Layer/services"
)

func SetupRoutes(db *database.Database) *http.ServeMux {
	// Initialize repository
	routerRepo := repository.NewRouterRepository(db.DB)
	
	// Initialize MikrotikService dengan repository
	ms := services.GetMikrotikService(routerRepo)
	
	// Initialize handlers
	routerHandler := handlers.NewRouterHandler(routerRepo)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", middleware.JSONMiddleware(handlers.HealthCheck))

	// ========== Router Management Routes ==========
	mux.HandleFunc("/api/routers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			middleware.JSONMiddleware(routerHandler.GetAllRouters)(w, r)
		case http.MethodPost:
			middleware.JSONMiddleware(routerHandler.CreateRouter)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/routers/active", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			middleware.JSONMiddleware(routerHandler.GetActiveRouters)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/routers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/routers/")
		parts := strings.Split(path, "/")

		if len(parts) == 1 && parts[0] != "" {
			switch r.Method {
			case http.MethodGet:
				middleware.JSONMiddleware(routerHandler.GetRouterByID)(w, r)
			case http.MethodPut:
				middleware.JSONMiddleware(routerHandler.UpdateRouter)(w, r)
			case http.MethodDelete:
				middleware.JSONMiddleware(routerHandler.DeleteRouter)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if len(parts) == 2 {
			if parts[1] == "status" && r.Method == http.MethodPatch {
				middleware.JSONMiddleware(routerHandler.UpdateRouterStatus)(w, r)
			} else if parts[1] == "active" && r.Method == http.MethodPatch {
				middleware.JSONMiddleware(routerHandler.SetActiveRouter)(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	// ========== Connection Management ==========
	mux.HandleFunc("/api/connections/status", middleware.JSONMiddleware(handlers.GetConnectionStatus(ms)))
	mux.HandleFunc("/api/connections/connect", middleware.JSONMiddleware(handlers.ConnectRouterHandler(ms)))
	mux.HandleFunc("/api/connections/disconnect", middleware.JSONMiddleware(handlers.DisconnectRouterHandler(ms)))

	// ========== Interface Routes (require router_id) ==========
	mux.HandleFunc("/api/interfaces", middleware.JSONMiddleware(handlers.GetInterfaces(ms)))
	mux.HandleFunc("/api/interfaces/enable", middleware.JSONMiddleware(handlers.EnableInterface(ms)))
	mux.HandleFunc("/api/interfaces/disable", middleware.JSONMiddleware(handlers.DisableInterface(ms)))

	// ========== Address Routes (require router_id) ==========
	mux.HandleFunc("/api/addresses", middleware.JSONMiddleware(handlers.GetAddresses(ms)))
	mux.HandleFunc("/api/addresses/add", middleware.JSONMiddleware(handlers.AddAddress(ms)))
	mux.HandleFunc("/api/addresses/remove", middleware.JSONMiddleware(handlers.RemoveAddress(ms)))

	// ========== Queue Routes (require router_id) ==========
	mux.HandleFunc("/api/queues", middleware.JSONMiddleware(handlers.GetQueues(ms)))
	mux.HandleFunc("/api/queues/add", middleware.JSONMiddleware(handlers.AddQueue(ms)))
	mux.HandleFunc("/api/queues/remove", middleware.JSONMiddleware(handlers.RemoveQueue(ms)))
	

	log.Println("✓ Routes configured successfully")
	return mux
}