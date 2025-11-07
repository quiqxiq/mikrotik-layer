package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"Mikrotik-Layer/models"
)

type RouterRepository struct {
	db *sql.DB
}

func NewRouterRepository(db *sql.DB) *RouterRepository {
	return &RouterRepository{db: db}
}

// Create - Tambah router baru
func (r *RouterRepository) Create(req *models.RouterCreateRequest) (*models.Router, error) {
	query := `
		INSERT INTO routers (name, hostname, username, password, keepalive, timeout, port, location, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	keepalive := true
	if req.Keepalive != nil {
		keepalive = *req.Keepalive
	}

	timeout := 300000
	if req.Timeout != nil {
		timeout = *req.Timeout
	}

	port := 8728
	if req.Port != nil {
		port = *req.Port
	}

	result, err := r.db.Exec(query, req.Name, req.Hostname, req.Username, req.Password,
		keepalive, timeout, port, req.Location, req.Description)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetByID(int(id))
}

// GetAll - Ambil semua router
func (r *RouterRepository) GetAll() ([]*models.Router, error) {
	query := `SELECT * FROM routers ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routers []*models.Router
	for rows.Next() {
		router := &models.Router{}
		err := rows.Scan(
			&router.ID, &router.UUID, &router.Name, &router.Hostname,
			&router.Username, &router.Password, &router.Keepalive, &router.Timeout,
			&router.Port, &router.Location, &router.Description, &router.IsActive,
			&router.LastSeen, &router.Status, &router.Version, &router.Uptime,
			&router.CreatedAt, &router.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		routers = append(routers, router)
	}

	return routers, nil
}

// GetByID - Ambil router by ID
func (r *RouterRepository) GetByID(id int) (*models.Router, error) {
	query := `SELECT * FROM routers WHERE id = ?`

	router := &models.Router{}
	err := r.db.QueryRow(query, id).Scan(
		&router.ID, &router.UUID, &router.Name, &router.Hostname,
		&router.Username, &router.Password, &router.Keepalive, &router.Timeout,
		&router.Port, &router.Location, &router.Description, &router.IsActive,
		&router.LastSeen, &router.Status, &router.Version, &router.Uptime,
		&router.CreatedAt, &router.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("router not found")
		}
		return nil, err
	}

	return router, nil
}

// GetByUUID - Ambil router by UUID
func (r *RouterRepository) GetByUUID(uuid string) (*models.Router, error) {
	query := `SELECT * FROM routers WHERE uuid = ?`

	router := &models.Router{}
	err := r.db.QueryRow(query, uuid).Scan(
		&router.ID, &router.UUID, &router.Name, &router.Hostname,
		&router.Username, &router.Password, &router.Keepalive, &router.Timeout,
		&router.Port, &router.Location, &router.Description, &router.IsActive,
		&router.LastSeen, &router.Status, &router.Version, &router.Uptime,
		&router.CreatedAt, &router.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("router not found")
		}
		return nil, err
	}

	return router, nil
}

// GetActiveRouters - Ambil router yang aktif
func (r *RouterRepository) GetActiveRouters() ([]*models.Router, error) {
	query := `SELECT * FROM routers WHERE is_active = true ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routers []*models.Router
	for rows.Next() {
		router := &models.Router{}
		err := rows.Scan(
			&router.ID, &router.UUID, &router.Name, &router.Hostname,
			&router.Username, &router.Password, &router.Keepalive, &router.Timeout,
			&router.Port, &router.Location, &router.Description, &router.IsActive,
			&router.LastSeen, &router.Status, &router.Version, &router.Uptime,
			&router.CreatedAt, &router.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		routers = append(routers, router)
	}

	return routers, nil
}

// Update - Update router
func (r *RouterRepository) Update(id int, req *models.RouterUpdateRequest) (*models.Router, error) {
	// Build dynamic update query
	var updates []string
	var args []interface{}

	if req.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Hostname != nil {
		updates = append(updates, "hostname = ?")
		args = append(args, *req.Hostname)
	}
	if req.Username != nil {
		updates = append(updates, "username = ?")
		args = append(args, *req.Username)
	}
	if req.Password != nil {
		updates = append(updates, "password = ?")
		args = append(args, *req.Password)
	}
	if req.Keepalive != nil {
		updates = append(updates, "keepalive = ?")
		args = append(args, *req.Keepalive)
	}
	if req.Timeout != nil {
		updates = append(updates, "timeout = ?")
		args = append(args, *req.Timeout)
	}
	if req.Port != nil {
		updates = append(updates, "port = ?")
		args = append(args, *req.Port)
	}
	if req.Location != nil {
		updates = append(updates, "location = ?")
		args = append(args, *req.Location)
	}
	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	}
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, *req.IsActive)
	}

	if len(updates) == 0 {
		return r.GetByID(id)
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE routers SET %s WHERE id = ?", strings.Join(updates, ", "))

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

// UpdateStatus - Update status router
func (r *RouterRepository) UpdateStatus(id int, status *models.RouterStatusUpdate) error {
	query := `
		UPDATE routers 
		SET status = ?, version = ?, uptime = ?, last_seen = ?, updated_at = ?
		WHERE id = ?
	`

	lastSeen := time.Now()
	if status.LastSeen != nil {
		lastSeen = *status.LastSeen
	}

	_, err := r.db.Exec(query, status.Status, status.Version, status.Uptime, lastSeen, time.Now(), id)
	return err
}

// SetActive - Set router sebagai aktif/non-aktif
func (r *RouterRepository) SetActive(id int, isActive bool) error {
	query := `UPDATE routers SET is_active = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, isActive, time.Now(), id)
	return err
}

// Delete - Hapus router
func (r *RouterRepository) Delete(id int) error {
	query := `DELETE FROM routers WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("router not found")
	}

	return nil
}

// GetByStatus - Ambil router by status
func (r *RouterRepository) GetByStatus(status string) ([]*models.Router, error) {
	query := `SELECT * FROM routers WHERE status = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routers []*models.Router
	for rows.Next() {
		router := &models.Router{}
		err := rows.Scan(
			&router.ID, &router.UUID, &router.Name, &router.Hostname,
			&router.Username, &router.Password, &router.Keepalive, &router.Timeout,
			&router.Port, &router.Location, &router.Description, &router.IsActive,
			&router.LastSeen, &router.Status, &router.Version, &router.Uptime,
			&router.CreatedAt, &router.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		routers = append(routers, router)
	}

	return routers, nil
}
