package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/db"
)

func RegisterUsersRoutes(r *gin.RouterGroup) {
	userRepo := repositories.NewUserRepository(db.GetDB())
	userHandler := handlers.NewUserHandler(userRepo)

	users := r.Group("/users")
	{
		users.POST("/register", userHandler.Register)
		users.POST("/login", userHandler.Login)
		users.POST("/logout", userHandler.Logout)
	}
}
