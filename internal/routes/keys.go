package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
)

func RegisterKeysRoutes(r *gin.RouterGroup) {
	keys := r.Group("/keys")
	keys.Use(middlewares.SessionMiddleware())
	{
		keys.GET("/", handlers.FindAllKeys)
		keys.GET("/:keyID", handlers.FindKeyByID)
		keys.POST("/generate-key-pair", handlers.GenerateKeyPair)
		keys.POST("/", handlers.CreateKey)
	}
}
