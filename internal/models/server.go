package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type LogType string

const (
	LogTypeSystem LogType = "Mo-SH"
	LogTypeInfo   LogType = "Info"
	LogTypeError  LogType = "Error"
)

type DockerInstallationLog struct {
	Type        LogType    `json:"type" db:"type"`
	Content     string     `json:"content" db:"content"`
	ProcessedAt *time.Time `json:"processedAt,omitempty" db:"processed_at"`
	IsLast      *bool      `json:"isLast,omitempty" db:"is_last"`
}

type DockerInstallationLogs []DockerInstallationLog

func (d *DockerInstallationLogs) Scan(src any) error {
	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", src)
	}

	var logs []DockerInstallationLog
	if err := json.Unmarshal(bytes, &logs); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}
	*d = logs
	return nil
}

func (d DockerInstallationLogs) Value() (driver.Value, error) {
	if d != nil {
		// If the value is not nil, serialize it to store in PostgreSQL.
		return json.Marshal(d)
	}
	return nil, nil
}

type Server struct {
	ID                             string                 `json:"id" db:"id"`
	KeyID                          string                 `json:"keyId" db:"key_id"`
	Name                           string                 `json:"name" db:"name"`
	Hostname                       string                 `json:"hostname" db:"hostname"`
	Port                           int                    `json:"port" db:"port"`
	HasDocker                      bool                   `json:"hasDocker" db:"has_docker"`
	DockerInstallationLogs         DockerInstallationLogs `json:"dockerInstallationLogs" db:"docker_installation_logs"`
	IsDockerInstalltionTaskRunning bool                   `json:"isDockerInstallationTaskRunning" db:"is_docker_installation_task_running"`
	CreatedAt                      time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt                      time.Time              `json:"updatedAt" db:"updated_at"`
}

type CreateServer struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}
