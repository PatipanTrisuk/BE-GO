package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
