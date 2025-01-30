package models

import "time"

type PrivateKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Key        *string    `json:"key,omitempty"`
	Type       *string    `json:"type,omitempty"`
	IsExternal *bool      `json:"isExternal,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}
