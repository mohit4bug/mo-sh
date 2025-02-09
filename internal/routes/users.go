package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
)

func RegisterUsersRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.POST("/register", handlers.RegisterUser)
		users.POST("/login", handlers.LoginUser)
		users.DELETE("/logout", handlers.LogoutUser)
	}
}
