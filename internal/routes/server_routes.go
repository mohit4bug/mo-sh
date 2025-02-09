package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/db"
)

func RegisterServersRoutes(r *gin.RouterGroup) {
	serverRepo := repositories.NewServerRepository(db.GetDB())
	serverHandler := handlers.NewServerHandler(serverRepo)

	servers := r.Group("/servers")
	servers.Use(middlewares.SessionMiddleware())
	{
		servers.POST("/", serverHandler.Create)
		servers.GET("/", serverHandler.FindAll)
		servers.GET("/:serverID/queue-docker-install", serverHandler.QueueDockerInstall)
	}
}
