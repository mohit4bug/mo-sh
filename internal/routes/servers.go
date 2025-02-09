package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
)

func RegisterServersRoutes(r *gin.RouterGroup) {
	servers := r.Group("/servers")
	servers.Use(middlewares.SessionMiddleware())
	{
		servers.POST("/", handlers.CreateServer)
		servers.GET("/", handlers.FindAllServers)
		servers.GET("/:serverID/queue-docker-install", handlers.QueueDockerInstall)
	}
}
