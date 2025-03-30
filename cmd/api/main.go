package main

import (
	"context"

	"github.com/mohit4bug/mo-sh/internal/workers"
	"github.com/mohit4bug/mo-sh/pkg/api"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/redis"
	"github.com/mohit4bug/mo-sh/pkg/ws"
)

func main() {
	ctx := context.Background()

	db := db.NewDatabase()
	redisClient := redis.NewRedisClient()
	webSocketManager := ws.NewWebSocketManager()

	dockerWorker := workers.NewDockerInstallationWorker(db, redisClient, &ctx, webSocketManager)
	dockerWorker.Start(3)

	r := api.NewRouter(db, redisClient, &ctx, webSocketManager)

	r.Run(":8000")
}
