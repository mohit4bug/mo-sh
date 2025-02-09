package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/pkg/session"
)

func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		session, err := session.GetSession(sessionID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Set("session", session)
		c.Next()
	}
}
