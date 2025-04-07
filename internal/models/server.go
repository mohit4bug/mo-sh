package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type LogType string

const (
	LogTypeSystem LogType = "System"
	LogTypeInfo   LogType = "Info"
	LogTypeError  LogType = "Error"
)

type DockerInstallationLog struct {
	Type    LogType `json:"type" db:"type"`
	Content string  `json:"content" db:"content"`
}

func (d *DockerInstallationLog) Scan(src any) error {
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	}
	return json.Unmarshal(data, d)
}

type DockerInstallationLogs []DockerInstallationLog

func (l *DockerInstallationLogs) Scan(src any) error {
	return pq.GenericArray{A: l}.Scan(src)
}

func (l DockerInstallationLogs) Value() (driver.Value, error) {
	var values []string
	for _, log := range l {
		b, err := json.Marshal(log)
		if err != nil {
			return nil, err
		}
		values = append(values, string(b))
	}
	return pq.Array(values).Value()
}

type Server struct {
	ID                       string                 `json:"id" db:"id"`
	KeyID                    string                 `json:"keyId" db:"key_id"`
	Name                     string                 `json:"name" db:"name"`
	Hostname                 string                 `json:"hostname" db:"hostname"`
	Port                     int                    `json:"port" db:"port"`
	HasDocker                bool                   `json:"hasDocker" db:"has_docker"`
	DockerInstallationLogs   DockerInstallationLogs `json:"dockerInstallationLogs" db:"docker_installation_logs"`
	DockerInstallationTaskID *string                `json:"dockerInstallationTaskId" db:"docker_installation_task_id"`
	CreatedAt                time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt                time.Time              `json:"updatedAt" db:"updated_at"`
}

type CreateServer struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}
