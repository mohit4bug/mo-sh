package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/mohit4bug/mo-sh/pkg/ws"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *sqlx.DB, redisClient *redis.Client, ctx *context.Context, webSocketManager *ws.WebSocketManager) *gin.Engine {
	userRepository := NewUserRepository(db, redisClient, ctx)
	keyRepository := NewKeyRepository(db, redisClient, ctx)
	serverRepository := NewServerRepository(db, redisClient, ctx)
	sourceRepository := NewSourceRepository(db, redisClient, ctx)
	webhookRepository := NewWebhookRepository(db, redisClient, ctx)

	r := gin.Default()
	r.Use(middlewares.Cors())

	v1 := r.Group("/api/v1")
	{
		v1.POST("/login", userRepository.Login)
		v1.POST("/register", userRepository.Register)

		v1.POST("/keys", middlewares.Auth(redisClient, ctx), keyRepository.Create)
		v1.GET("/keys", middlewares.Auth(redisClient, ctx), keyRepository.FindAll)
		v1.GET("/keys/:keyID", middlewares.Auth(redisClient, ctx), keyRepository.FindByID)
		v1.POST("/keys/generate", middlewares.Auth(redisClient, ctx), keyRepository.GenerateKey)

		v1.POST("/servers", middlewares.Auth(redisClient, ctx), serverRepository.Create)
		v1.GET("/servers", middlewares.Auth(redisClient, ctx), serverRepository.FindAll)
		v1.GET("/servers/:serverID", middlewares.Auth(redisClient, ctx), serverRepository.FindByID)
		v1.GET("/servers/:serverID/queue-docker-install", middlewares.Auth(redisClient, ctx), serverRepository.QueueDockerInstall)

		v1.POST("/sources", middlewares.Auth(redisClient, ctx), sourceRepository.Create)
		v1.GET("/sources", middlewares.Auth(redisClient, ctx), sourceRepository.FindAll)
		v1.GET("/sources/:sourceID", middlewares.Auth(redisClient, ctx), sourceRepository.FindByID)
		v1.GET("/sources/:sourceID/register-github-app", middlewares.Auth(redisClient, ctx), sourceRepository.RegisterGithubApp)

		v1.GET("/webhooks/github/redirect", webhookRepository.HandleGithubRedirect)

		v1.GET("/ws/:connID", webSocketManager.ServeWebSocketRequests)
	}

	return r
}
