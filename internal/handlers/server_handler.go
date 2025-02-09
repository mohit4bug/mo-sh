package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/config"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/rmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ServerHandler struct {
	repo repositories.ServerRepository
}

func NewServerHandler(repo repositories.ServerRepository) *ServerHandler {
	return &ServerHandler{repo: repo}
}

type ServerInput struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}

func (h *ServerHandler) Create(c *gin.Context) {
	var input ServerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server := &models.Server{
		Name:     input.Name,
		Hostname: input.Hostname,
		Port:     input.Port,
		KeyID:    input.KeyID,
	}

	id, err := h.repo.Create(server)
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

func (h *ServerHandler) FindAll(c *gin.Context) {
	servers, err := h.repo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"servers": servers}})
}

func (h *ServerHandler) QueueDockerInstall(c *gin.Context) {
	serverID := c.Param("serverID")

	inProgress, err := h.repo.HasPendingInstallation(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if inProgress {
		c.JSON(http.StatusConflict, gin.H{"error": "Installation already in progress"})
		return
	}

	if err := h.repo.CreateDockerInstallation(serverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rmqChannel := rmq.GetRMQChannel()

	_, err = rmqChannel.QueueDeclare(config.DockerInstallationQueue, true, false, false, false, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = rmqChannel.Publish("", config.DockerInstallationQueue, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(serverID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}
