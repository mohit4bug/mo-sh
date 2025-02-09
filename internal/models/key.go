package models

import (
	"time"

	"github.com/mohit4bug/mo-sh/internal/shared"
)

type Key struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Key        string    `json:"key"`
	Type       string    `json:"type"`
	IsExternal bool      `json:"isExternal"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (k *Key) ExtractPublicKey() (string, error) {
	return shared.ExtractPublicKey(k.Key, k.Type)
}
