package models

import (
	"time"
)

type Key struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Key        string    `json:"key" db:"key"`
	IsExternal bool      `json:"isExternal" db:"is_external"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
}

type CreateKey struct {
	Name string `json:"name" binding:"required"`
	Key  string `json:"key" binding:"required"`
}

type GenerateKey struct {
	Type string `json:"type" binding:"required"`
}
