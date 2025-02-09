package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/config"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/rmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ServerInput struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}

func CreateServer(c *gin.Context) {
	var input ServerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := db.GetDB()
	var id string
	err := db.QueryRow("INSERT INTO servers (name, hostname, port, key_id) VALUES ($1, $2, $3, $4) RETURNING id", input.Name, input.Hostname, input.Port, input.KeyID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "OK",
		"data": gin.H{"server": gin.H{
			"id": id,
		}},
	})
}

func FindAllServers(c *gin.Context) {
	db := db.GetDB()

	rows, err := db.Query("SELECT id, name, hostname, port, key_id FROM servers")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	servers := make([]models.Server, 0)

	for rows.Next() {
		var server models.Server
		if err := rows.Scan(&server.ID, &server.Name, &server.Hostname, &server.Port, &server.KeyID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		servers = append(servers, server)
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"servers": servers}})
}

func QueueDockerInstall(c *gin.Context) {
	db := db.GetDB()
	serverID := c.Param("serverID")

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var inProgress bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 
			FROM docker_installations 
			WHERE server_id = $1 
			AND status IN ('not_started', 'in_progress')
		)`, serverID).Scan(&inProgress)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if inProgress {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "Installation already in progress"})
		return
	}

	_, err = tx.Exec("INSERT INTO docker_installations (server_id, status) VALUES ($1, 'not_started')", serverID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rmqChannel := rmq.GetRMQChannel()

	_, err = rmqChannel.QueueDeclare(config.DockerInstallationQueue, true, false, false, false, nil)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = rmqChannel.Publish("", config.DockerInstallationQueue, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(serverID),
	})
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}
