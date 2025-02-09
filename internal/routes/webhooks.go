package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
)

func RegisterWebhooksRoutes(r *gin.RouterGroup) {
	webhooks := r.Group("/webhooks")
	{
		webhooks.GET("/github/redirect", handlers.HandleGithubRedirect)
	}
}
