package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
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
	Ctx         *context.Context
}

func NewServerRepository(db *sqlx.DB, redisClient *redis.Client, ctx *context.Context) *serverRepository {
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
	_, err := r.DB.NamedExecContext(*r.Ctx, query, newServer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OK"})
}

func (r *serverRepository) FindAll(c *gin.Context) {
	var servers []models.Server = []models.Server{}
	err := r.DB.SelectContext(*r.Ctx, &servers, "select * from servers")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
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
	if err := r.DB.GetContext(*r.Ctx, &server, "select * from servers where id = $1", serverID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "404"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
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
	if err := r.DB.GetContext(*r.Ctx, &server, query, serverID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "NOT_FOUND"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if server.DockerInstallationTaskID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Docker installation already in progress."})
		return
	}

	sshClient := ssh.NewClient(server.Hostname, server.Port, "root", []byte(server.Key))
	if err := sshClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer sshClient.Close()

	if sshClient.CheckCommand("dockerr --version") {
		c.JSON(http.StatusOK, gin.H{"message": "Docker already installed."})
		return
	}

	taskID := uuid.New().String()

	updateQuery := `
		update servers
		set docker_installation_task_id = :task_id
		where id = :server_id
	`

	params := gin.H{
		"task_id":   taskID,
		"server_id": serverID,
	}

	if _, err := r.DB.NamedExecContext(*r.Ctx, updateQuery, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	task := fmt.Sprintf("%s:%s", serverID, taskID)

	if err := r.RedisClient.LPush(*r.Ctx, "docker_installation_queue", task).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Docker installation queued.",
		"data": gin.H{
			"taskId": taskID,
		},
	})
}
