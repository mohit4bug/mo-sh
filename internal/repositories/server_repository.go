package repositories

import (
	"database/sql"

	"github.com/mohit4bug/mo-sh/internal/models"
)

type ServerRepository interface {
	Create(server *models.Server) (string, error)
	FindAll() ([]models.Server, error)
	FindByID(id string) (*models.Server, error)
	HasPendingInstallation(serverID string) (bool, error)
	CreateDockerInstallation(serverID string) error
}

type serverRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) ServerRepository {
	return &serverRepository{db: db}
}

func (r *serverRepository) Create(server *models.Server) (string, error) {
	var id string
	err := r.db.QueryRow(
		"INSERT INTO servers (name, hostname, port, key_id) VALUES ($1, $2, $3, $4) RETURNING id",
		server.Name,
		server.Hostname,
		server.Port,
		server.KeyID,
	).Scan(&id)
	return id, err
}

func (r *serverRepository) FindAll() ([]models.Server, error) {
	rows, err := r.db.Query("SELECT id, name, hostname, port, key_id FROM servers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]models.Server, 0)
	for rows.Next() {
		var server models.Server
		if err := rows.Scan(&server.ID, &server.Name, &server.Hostname, &server.Port, &server.KeyID); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func (r *serverRepository) FindByID(id string) (*models.Server, error) {
	var server models.Server
	err := r.db.QueryRow(
		"SELECT id, name, hostname, port, key_id FROM servers WHERE id = $1",
		id,
	).Scan(&server.ID, &server.Name, &server.Hostname, &server.Port, &server.KeyID)
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func (r *serverRepository) HasPendingInstallation(serverID string) (bool, error) {
	var inProgress bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 
			FROM docker_installations 
			WHERE server_id = $1 
			AND status IN ('not_started', 'in_progress')
		)`, serverID).Scan(&inProgress)
	return inProgress, err
}

func (r *serverRepository) CreateDockerInstallation(serverID string) error {
	_, err := r.db.Exec(
		"INSERT INTO docker_installations (server_id, status) VALUES ($1, 'not_started')",
		serverID,
	)
	return err
}
