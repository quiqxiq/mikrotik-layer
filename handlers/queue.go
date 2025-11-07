package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"Mikrotik-Layer/models"
	"Mikrotik-Layer/services"
)

func GetQueues(ms *services.MikrotikService) http.HandlerFunc {
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

		queues, err := ms.GetQueues(routerID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Data:    queues,
		})
	}
}

func AddQueue(ms *services.MikrotikService) http.HandlerFunc {
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

		name := r.URL.Query().Get("name")
		target := r.URL.Query().Get("target")
		maxLimit := r.URL.Query().Get("max-limit")

		if name == "" || target == "" || maxLimit == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'name', 'target', dan 'max-limit' diperlukan",
			})
			return
		}

		err = ms.AddQueue(routerID, name, target, maxLimit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Message: "Queue berhasil ditambahkan",
		})
	}
}

func RemoveQueue(ms *services.MikrotikService) http.HandlerFunc {
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

		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'id' diperlukan",
			})
			return
		}

		err = ms.RemoveQueue(routerID, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Message: "Queue berhasil dihapus",
		})
	}
}

// ==================== handlers/traffic_handler.go (UPDATED) ====================
