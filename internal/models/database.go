package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type DatabasePlan struct {
	ID          string    `json:"id" db:"id"`
	Engine      string    `json:"engine" db:"engine"`
	Version     string    `json:"version" db:"version"`
	Image       string    `json:"image" db:"image"`
	DefaultPort int       `json:"defaultPort" db:"default_port"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

type DatabaseDeploymentLog struct {
	Type        LogType    `json:"type" db:"type"`
	Content     string     `json:"content" db:"content"`
	ProcessedAt *time.Time `json:"processedAt,omitempty" db:"processed_at"`
	IsLast      *bool      `json:"isLast,omitempty" db:"is_last"`
}

type DatabaseDeploymentLogs []DatabaseDeploymentLog

func (d *DatabaseDeploymentLogs) Scan(src any) error {
	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", src)
	}

	var logs []DatabaseDeploymentLog
	if err := json.Unmarshal(bytes, &logs); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}
	*d = logs
	return nil
}

func (d DatabaseDeploymentLogs) Value() (driver.Value, error) {
	if d != nil {
		return json.Marshal(d)
	}
	return nil, nil
}

type DatabaseService struct {
	ID               string                 `json:"id" db:"id"`
	UserID           string                 `json:"userId" db:"user_id"`
	ServerID         string                 `json:"serverId" db:"server_id"`
	PlanID           string                 `json:"planId" db:"plan_id"`
	Name             string                 `json:"name" db:"name"`
	DatabaseName     string                 `json:"databaseName" db:"database_name"`
	DatabaseUser     string                 `json:"databaseUser" db:"database_user"`
	DatabasePassword string                 `json:"databasePassword,omitempty" db:"database_password"`
	ContainerName    string                 `json:"containerName" db:"container_name"`
	VolumeName       string                 `json:"volumeName" db:"volume_name"`
	NetworkName      string                 `json:"networkName" db:"network_name"`
	InternalPort     int                    `json:"internalPort" db:"internal_port"`
	PublicPort       sql.NullInt64          `json:"-" db:"public_port"`
	ExposePublicly   bool                   `json:"exposePublicly" db:"expose_publicly"`
	Status           string                 `json:"status" db:"status"`
	DeploymentLogs   DatabaseDeploymentLogs `json:"deploymentLogs" db:"deployment_logs"`
	CreatedAt        time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time              `json:"updatedAt" db:"updated_at"`
}

type DatabaseServiceResponse struct {
	ID                 string    `json:"id"`
	ServerID           string    `json:"serverId"`
	PlanID             string    `json:"planId"`
	Name               string    `json:"name"`
	DatabaseName       string    `json:"databaseName"`
	DatabaseUser       string    `json:"databaseUser"`
	ContainerName      string    `json:"containerName"`
	NetworkName        string    `json:"networkName"`
	InternalPort       int       `json:"internalPort"`
	PublicPort         *int      `json:"publicPort"`
	ExposePublicly     bool      `json:"exposePublicly"`
	Status             string    `json:"status"`
	InternalConnection string    `json:"internalConnection"`
	PublicConnection   *string   `json:"publicConnection"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type CreateDatabaseService struct {
	Engine         string `json:"engine" binding:"required"`
	Version        string `json:"version" binding:"required"`
	Name           string `json:"name" binding:"required"`
	DatabaseName   string `json:"databaseName" binding:"required"`
	DatabaseUser   string `json:"databaseUser" binding:"required"`
	ExposePublicly bool   `json:"exposePublicly"`
	PublicPort     *int   `json:"publicPort"`
}

type DatabaseDeploymentJob struct {
	ID                string         `json:"id" db:"id"`
	DatabaseServiceID string         `json:"databaseServiceId" db:"database_service_id"`
	Status            string         `json:"status" db:"status"`
	ErrorMessage      sql.NullString `json:"-" db:"error_message"`
	StartedAt         sql.NullTime   `json:"-" db:"started_at"`
	FinishedAt        sql.NullTime   `json:"-" db:"finished_at"`
	CreatedAt         time.Time      `json:"createdAt" db:"created_at"`
}

func NewDatabaseServiceResponse(service DatabaseService, serverHost string) DatabaseServiceResponse {
	var publicPort *int
	var publicConnection *string
	if service.PublicPort.Valid {
		port := int(service.PublicPort.Int64)
		publicPort = &port
		conn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", service.DatabaseUser, service.DatabasePassword, serverHost, port, service.DatabaseName)
		publicConnection = &conn
	}

	return DatabaseServiceResponse{
		ID:                 service.ID,
		ServerID:           service.ServerID,
		PlanID:             service.PlanID,
		Name:               service.Name,
		DatabaseName:       service.DatabaseName,
		DatabaseUser:       service.DatabaseUser,
		ContainerName:      service.ContainerName,
		NetworkName:        service.NetworkName,
		InternalPort:       service.InternalPort,
		PublicPort:         publicPort,
		ExposePublicly:     service.ExposePublicly,
		Status:             service.Status,
		InternalConnection: fmt.Sprintf("postgres://%s:%s@%s:%d/%s", service.DatabaseUser, service.DatabasePassword, service.ContainerName, service.InternalPort, service.DatabaseName),
		PublicConnection:   publicConnection,
		CreatedAt:          service.CreatedAt,
		UpdatedAt:          service.UpdatedAt,
	}
}
