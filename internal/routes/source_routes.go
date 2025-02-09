package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/db"
)

func RegisterSourcesRoutes(r *gin.RouterGroup) {
	sourceRepo := repositories.NewSourceRepository(db.GetDB())
	sourceHandler := handlers.NewSourceHandler(sourceRepo)

	sources := r.Group("/sources")
	sources.Use(middlewares.SessionMiddleware())
	{
		sources.POST("/", sourceHandler.Create)
		sources.GET("/", sourceHandler.FindAll)
		sources.GET("/:sourceID/register-github-app", sourceHandler.RegisterGithubApp)
	}
}
