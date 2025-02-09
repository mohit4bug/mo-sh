package models

import (
	"time"

	"github.com/mohit4bug/mo-sh/internal/shared"
)

type Key struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Key        string     `json:"key,omitempty"`
	Type       string     `json:"type,omitempty"`
	IsExternal bool       `json:"isExternal,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}

func (k *Key) ExtractPublicKey() (string, error) {
	return shared.ExtractPublicKey(k.Key, k.Type)
}
