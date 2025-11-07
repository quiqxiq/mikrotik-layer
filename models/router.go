// ==================== models/router.go ====================
package models

import (
	"time"
)

type Router struct {
	ID          int       `json:"id" db:"id"`
	UUID        string    `json:"uuid" db:"uuid"`
	Name        string    `json:"name" db:"name"`
	Hostname    string    `json:"hostname" db:"hostname"`
	Username    string    `json:"username" db:"username"`
	Password    string    `json:"password" db:"password"`
	Keepalive   bool      `json:"keepalive" db:"keepalive"`
	Timeout     int       `json:"timeout" db:"timeout"`
	Port        int       `json:"port" db:"port"`
	Location    *string   `json:"location,omitempty" db:"location"`
	Description *string   `json:"description,omitempty" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	LastSeen    *time.Time `json:"last_seen,omitempty" db:"last_seen"`
	Status      string    `json:"status" db:"status"` // online, offline, error
	Version     *string   `json:"version,omitempty" db:"version"`
	Uptime      *string   `json:"uptime,omitempty" db:"uptime"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type RouterCreateRequest struct {
	Name        string  `json:"name" binding:"required"`
	Hostname    string  `json:"hostname" binding:"required"`
	Username    string  `json:"username" binding:"required"`
	Password    string  `json:"password" binding:"required"`
	Keepalive   *bool   `json:"keepalive,omitempty"`
	Timeout     *int    `json:"timeout,omitempty"`
	Port        *int    `json:"port,omitempty"`
	Location    *string `json:"location,omitempty"`
	Description *string `json:"description,omitempty"`
}

type RouterUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Hostname    *string `json:"hostname,omitempty"`
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"`
	Keepalive   *bool   `json:"keepalive,omitempty"`
	Timeout     *int    `json:"timeout,omitempty"`
	Port        *int    `json:"port,omitempty"`
	Location    *string `json:"location,omitempty"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type RouterStatusUpdate struct {
	Status   string     `json:"status"`
	Version  *string    `json:"version,omitempty"`
	Uptime   *string    `json:"uptime,omitempty"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}
