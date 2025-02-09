package routes

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	RegisterKeysRoutes(api)
	RegisterServersRoutes(api)
	RegisterSourcesRoutes(api)
	RegisterUsersRoutes(api)
	RegisterWebhooksRoutes(api)
}
