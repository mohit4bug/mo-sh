package main

import (
	"context"
	"log"

	"github.com/mohit4bug/mo-sh/internal/config"
	"github.com/mohit4bug/mo-sh/internal/workers"
	"github.com/mohit4bug/mo-sh/pkg/api"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/redis"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db := db.NewDatabase(cfg.DatabaseURL)
	redisClient := redis.NewRedisClient(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)

	dockerWorker := workers.NewDockerInstallationWorker(db, redisClient, ctx)
	dockerWorker.Start(cfg.WorkerCount)

	r, err := api.NewRouter(db, redisClient, ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatal(err)
	}
}
