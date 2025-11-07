// ==================== handlers/interface_handler.go (UPDATED) ====================
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"Mikrotik-Layer/models"
	"Mikrotik-Layer/services"
)

func GetInterfaces(ms *services.MikrotikService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: true,
			Data:    interfaces,
		})
	}
}

func EnableInterface(ms *services.MikrotikService) http.HandlerFunc {
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
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'name' diperlukan",
			})
			return
		}

		err = ms.EnableInterface(routerID, name)
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
			Message: "Interface diaktifkan",
		})
	}
}

func DisableInterface(ms *services.MikrotikService) http.HandlerFunc {
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
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ApiResponse{
				Success: false,
				Error:   "parameter 'name' diperlukan",
			})
			return
		}

		err = ms.DisableInterface(routerID, name)
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
			Message: "Interface dinonaktifkan",
		})
	}
}

// ==================== handlers/address_handler.go (UPDATED) ====================
