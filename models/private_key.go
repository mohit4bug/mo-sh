package models

import "time"

type PrivateKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Key        *string    `json:"key,omitempty"`
	IsExternal *bool      `json:"isExternal,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}
