package middlewares

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/pkg/session"
	"github.com/redis/go-redis/v9"
)

func Auth(redisClient *redis.Client, ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		sessionStore := session.NewSessionStore(redisClient, ctx)
		session, err := sessionStore.FindByID(sessionID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Set("session", session)
		c.Next()
	}
}
