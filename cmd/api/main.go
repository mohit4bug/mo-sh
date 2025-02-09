package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/middlewares"
	"github.com/mohit4bug/mo-sh/internal/routes"
	"github.com/mohit4bug/mo-sh/internal/workers"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
	"github.com/mohit4bug/mo-sh/pkg/rmq"
)

func main() {

	r := gin.Default()

	r.Use(middlewares.CorsMiddleware())
	r.Use(middlewares.RateLimitMiddleware(500, 1*time.Minute))

	db.InitDB()
	defer db.CloseDB()

	db := db.GetDB()
	defer db.Close()

	rdb.InitRedis()
	defer rdb.CloseRedis()

	rmq.InitRMQ()
	defer rmq.CloseRMQ()

	rmqChannel := rmq.GetRMQChannel()
	defer rmqChannel.Close()

	routes.SetupRoutes(r)

	workers.ListenForDockerInstallationEvents(rmqChannel, db, 4)

	r.Run(":8000")
}
