package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/config"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *sqlx.DB, redisClient *redis.Client, ctx context.Context, cfg config.Config) (*gin.Engine, error) {
	userRepository := NewUserRepository(db, redisClient, ctx, cfg)
	keyRepository := NewKeyRepository(db, redisClient, ctx)
	serverRepository := NewServerRepository(db, redisClient, ctx)
	sourceRepository := NewSourceRepository(db, redisClient, ctx, cfg)
	webhookRepository := NewWebhookRepository(db, redisClient, ctx, cfg)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middlewares.Cors(cfg.ClientURL))
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

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
		v1.GET("/servers/:serverID/pending-logs", middlewares.Auth(redisClient, ctx), serverRepository.GetPendingLogs)
		v1.GET("/servers/:serverID/queue-caddy-install", middlewares.Auth(redisClient, ctx), serverRepository.QueueCaddyInstall)
		v1.GET("/servers/:serverID/caddy-pending-logs", middlewares.Auth(redisClient, ctx), serverRepository.GetCaddyPendingLogs)

		v1.POST("/sources", middlewares.Auth(redisClient, ctx), sourceRepository.Create)
		v1.GET("/sources", middlewares.Auth(redisClient, ctx), sourceRepository.FindAll)
		v1.GET("/sources/:sourceID", middlewares.Auth(redisClient, ctx), sourceRepository.FindByID)
		v1.GET("/sources/:sourceID/register-github-app", middlewares.Auth(redisClient, ctx), sourceRepository.RegisterGithubApp)

		v1.GET("/webhooks/github/redirect", webhookRepository.HandleGithubRedirect)

	}

	return r, nil
}
