package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/handlers"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/mohit4bug/mo-sh/internal/routes"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
)

func main() {

	r := gin.Default()

	r.Use(middlewares.CorsMiddleware())
	r.Use(middlewares.RateLimitMiddleware(500, 1*time.Minute))

	db.InitDB()
	defer db.CloseDB()

	rdb.InitRedis()
	defer rdb.CloseRedis()

	routes.SetupRoutes(r)

	go handlers.ListenForDockerInstall()

	r.Run()
}
