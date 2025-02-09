package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/db"
)

func RegisterWebhooksRoutes(r *gin.RouterGroup) {
	webhookRepo := repositories.NewWebhookRepository(db.GetDB())
	webhookHandler := handlers.NewWebhookHandler(webhookRepo)

	webhooks := r.Group("/webhooks")
	{
		webhooks.GET("/github/redirect", webhookHandler.HandleGithubRedirect)
	}
}
