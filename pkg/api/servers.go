package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/internal/workers"
	"github.com/mohit4bug/mo-sh/pkg/ssh"
	"github.com/redis/go-redis/v9"
)

type ServerRepository interface {
	Create(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	QueueDockerInstall(c *gin.Context)
}

type serverRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
}

func NewServerRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *serverRepository {
	return &serverRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}

func (r *serverRepository) Create(c *gin.Context) {
	var input models.CreateServer
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newServer := &models.Server{
		Name:     input.Name,
		Hostname: input.Hostname,
		Port:     input.Port,
		KeyID:    input.KeyID,
	}

	query := `INSERT INTO servers (name, hostname, port, key_id) VALUES (:name, :hostname, :port, :key_id)`
	_, err := r.DB.NamedExecContext(r.Ctx, query, newServer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OK"})
}

func (r *serverRepository) FindAll(c *gin.Context) {
	var servers []models.Server = []models.Server{}
	err := r.DB.SelectContext(r.Ctx, &servers, "select * from servers")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"servers": servers},
	})
}

func (r *serverRepository) FindByID(c *gin.Context) {
	serverID := c.Param("serverID")

	var server models.Server
	if err := r.DB.GetContext(r.Ctx, &server, "select * from servers where id = $1", serverID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "404"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"server": server},
	})
}

func (r *serverRepository) QueueDockerInstall(c *gin.Context) {
	serverID := c.Param("serverID")

	type ServerWithKey struct {
		models.Server
		Key string `db:"key"`
	}

	query := `
		select
			s.*, k.key
		from
			servers s
		inner join
			keys k ON s.key_id = k.id
		where
			s.id = $1
	`

	var server ServerWithKey
	if err := r.DB.GetContext(r.Ctx, &server, query, serverID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": shared.ErrNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	if server.IsDockerInstalltionTaskRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Docker installation already in progress. Please wait."})
		return
	}

	sshClient := ssh.NewClient(server.Hostname, server.Port, "root", []byte(server.Key))
	if err := sshClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrSSHConnection})
		return
	}
	defer sshClient.Close()

	if sshClient.CheckCommand("dockerr --version") {
		c.JSON(http.StatusOK, gin.H{"message": "Docker is already installed."})
		return
	}

	updateQuery := `
		update servers
		set is_docker_installation_task_running = :isDockerInstallationTaskRunning
		where id = :serverID
	`

	params := gin.H{
		"isDockerInstallationTaskRunning": true,
		"serverID":                        serverID,
	}

	if _, err := r.DB.NamedExecContext(r.Ctx, updateQuery, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	if err := r.RedisClient.LPush(r.Ctx, workers.DockerPendingQueue, serverID).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Docker installation queued."})
}

func (r *serverRepository) GetPendingLogs(c *gin.Context) {
	serverID := c.Param("serverID")

	tx, err := r.DB.BeginTx(r.Ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}
	defer tx.Rollback()

	var exists bool
	if err = tx.QueryRowContext(r.Ctx, `select exists(select 1 from servers where id = $1)`, serverID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": shared.ErrNotFound})
		return
	}

	var logs models.DockerInstallationLogs
	query := `
		select 
			coalesce(jsonb_agg(logs), '[]'::jsonb) as unprocessed_logs
		from (
			select 
				jsonb_array_elements(coalesce(docker_installation_logs, '[]'::jsonb)) as logs
			from 
				servers
			where 
				id = $1
		) as logs
		where 
			logs->>'processedAt' is null
	`

	if err = tx.QueryRowContext(r.Ctx, query, serverID).Scan(&logs); err != nil {
		if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
			return
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	updateQuery := `
		update servers
		set docker_installation_logs = coalesce((
			select jsonb_agg(
				case
					when elem->>'processedAt' is null 
					then jsonb_set(elem, '{processedAt}', to_jsonb($1::text))
					else elem
				end
			)
			from jsonb_array_elements(docker_installation_logs) as elem
			--- handle null docker_installation_logs by defaulting to an empty array (typically on first run)
		), '[]'::jsonb)
		where id = $2
	`

	_, err = tx.ExecContext(r.Ctx, updateQuery, now, serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data": gin.H{
			"logs": logs,
		},
	})
}
