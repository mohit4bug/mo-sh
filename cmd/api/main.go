package main

import (
	"context"

	"github.com/mohit4bug/mo-sh/internal/workers"
	"github.com/mohit4bug/mo-sh/pkg/api"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/redis"
)

func main() {
	ctx := context.Background()

	db := db.NewDatabase()
	redisClient := redis.NewRedisClient()

	dockerWorker := workers.NewDockerInstallationWorker(db, redisClient, ctx)
	dockerWorker.Start(3)

	r := api.NewRouter(db, redisClient, ctx)

	r.Run(":8000")
}
