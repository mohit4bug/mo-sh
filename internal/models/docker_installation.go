package models

import "time"

type DockerInstallation struct {
	ID        string     `json:"id"`
	ServerID  string     `json:"serverId"`
	Status    string     `json:"status"`
	Logs      []string   `json:"logs"` // NOTE: This can be made more advanced if needed, rather than just an array of strings.
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}
