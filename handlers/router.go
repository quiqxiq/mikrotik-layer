package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"Mikrotik-Layer/models"
	"Mikrotik-Layer/repository"
)

type RouterHandler struct {
	repo *repository.RouterRepository
}

func NewRouterHandler(repo *repository.RouterRepository) *RouterHandler {
	return &RouterHandler{repo: repo}
}

// CreateRouter - POST /api/routers
func (h *RouterHandler) CreateRouter(w http.ResponseWriter, r *http.Request) {
	var req models.RouterCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	router, err := h.repo.Create(&req)
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
		Message: "Router berhasil ditambahkan",
		Data:    router,
	})
}

// GetAllRouters - GET /api/routers
func (h *RouterHandler) GetAllRouters(w http.ResponseWriter, r *http.Request) {
	routers, err := h.repo.GetAll()
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
		Data:    routers,
	})
}

// GetRouterByID - GET /api/routers/{id}
func (h *RouterHandler) GetRouterByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/routers/")
	id, err := strconv.Atoi(path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid router ID",
		})
		return
	}

	router, err := h.repo.GetByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(models.ApiResponse{
		Success: true,
		Data:    router,
	})
}

// GetActiveRouters - GET /api/routers/active
func (h *RouterHandler) GetActiveRouters(w http.ResponseWriter, r *http.Request) {
	routers, err := h.repo.GetActiveRouters()
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
		Data:    routers,
	})
}

// UpdateRouter - PUT /api/routers/{id}
func (h *RouterHandler) UpdateRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/routers/")
	id, err := strconv.Atoi(path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid router ID",
		})
		return
	}

	var req models.RouterUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	router, err := h.repo.Update(id, &req)
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
		Message: "Router berhasil diupdate",
		Data:    router,
	})
}

// UpdateRouterStatus - PATCH /api/routers/{id}/status
func (h *RouterHandler) UpdateRouterStatus(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/routers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid URL",
		})
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid router ID",
		})
		return
	}

	var req models.RouterStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.repo.UpdateStatus(id, &req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(models.ApiResponse{
		Success: true,
		Message: "Status router berhasil diupdate",
	})
}

// SetActiveRouter - PATCH /api/routers/{id}/active
func (h *RouterHandler) SetActiveRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/routers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid URL",
		})
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid router ID",
		})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.repo.SetActive(id, req.IsActive); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	status := "diaktifkan"
	if !req.IsActive {
		status = "dinonaktifkan"
	}

	json.NewEncoder(w).Encode(models.ApiResponse{
		Success: true,
		Message: "Router berhasil " + status,
	})
}

// DeleteRouter - DELETE /api/routers/{id}
func (h *RouterHandler) DeleteRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/routers/")
	id, err := strconv.Atoi(path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   "Invalid router ID",
		})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ApiResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(models.ApiResponse{
		Success: true,
		Message: "Router berhasil dihapus",
	})
}