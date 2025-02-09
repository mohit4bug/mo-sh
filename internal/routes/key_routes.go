package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/db"
)

func RegisterKeysRoutes(r *gin.RouterGroup) {
	keyRepo := repositories.NewKeyRepository(db.GetDB())
	keyHandler := handlers.NewKeyHandler(keyRepo)

	keys := r.Group("/keys")
	keys.Use(middlewares.SessionMiddleware())
	{
		keys.POST("/", keyHandler.Create)
		keys.GET("/", keyHandler.FindAll)
		keys.GET("/:keyID", keyHandler.FindByID)
		keys.POST("/generate", keyHandler.GenerateKeyPair)
	}
}
