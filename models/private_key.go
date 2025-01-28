package models

type PrivateKey struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Key        string `json:"key"`
	IsExternal bool   `json:"is_external"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}
