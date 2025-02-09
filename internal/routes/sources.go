package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
)

func RegisterSourcesRoutes(r *gin.RouterGroup) {
	sources := r.Group("/sources")

	sources.Use(middlewares.SessionMiddleware())
	{
		sources.POST("/", handlers.CreateSource)
		sources.GET("/", handlers.FindAllSources)
		sources.GET("/:sourceID/register-github-app", handlers.RegisterGithubApp)
	}
}
