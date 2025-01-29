package models

import "time"

type PrivateKey struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Key        string    `json:"key"`
	IsExternal bool      `json:"is_external"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
