package models

import (
	"time"

	"github.com/lib/pq"
)

type Server struct {
	ID                       string         `json:"id" db:"id"`
	KeyID                    string         `json:"keyId" db:"key_id"`
	Name                     string         `json:"name" db:"name"`
	Hostname                 string         `json:"hostname" db:"hostname"`
	Port                     int            `json:"port" db:"port"`
	HasDocker                bool           `json:"hasDocker" db:"has_docker"`
	DockerInstallationLogs   pq.StringArray `json:"dockerInstallationLogs" db:"docker_installation_logs"`
	DockerInstallationTaskID *string        `json:"dockerInstallationTaskId" db:"docker_installation_task_id"`
	CreatedAt                time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt                time.Time      `json:"updatedAt" db:"updated_at"`
}

type CreateServer struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}
