package models

import "time"

type PrivateKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Key        *string    `json:"key,omitempty"`
	IsExternal bool       `json:"isExternal"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}
