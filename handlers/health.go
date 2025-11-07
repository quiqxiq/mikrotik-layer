package handlers

import (
	"encoding/json"
	"net/http"

	"Mikrotik-Layer/models"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(models.ApiResponse{
		Success: true,
		Message: "API berjalan normal",
	})
}