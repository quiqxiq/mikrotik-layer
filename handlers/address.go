package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"Mikrotik-Layer/models"
	"Mikrotik-Layer/services"
)

func GetAddresses(ms *services.MikrotikService) http.HandlerFunc {
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

		addresses, err := ms.GetAddresses(routerID)
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
			Data:    addresses,
		})
	}
}

func AddAddress(ms *services.MikrotikService) http.HandlerFunc {
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

		iface := r.URL.Query().Get("interface")
		address := r.URL.Query().Get("address")

		if iface == "" || address == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'interface' dan 'address' diperlukan",
			})
			return
		}

		err = ms.AddAddress(routerID, iface, address)
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
			Message: "Address berhasil ditambahkan",
		})
	}
}

func RemoveAddress(ms *services.MikrotikService) http.HandlerFunc {
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

		err = ms.RemoveAddress(routerID, id)
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
			Message: "Address berhasil dihapus",
		})
	}
}

// ==================== handlers/queue_handler.go (UPDATED) ====================
