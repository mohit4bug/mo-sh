package api

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type DatabaseRepository interface{}

type databaseRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
}

func NewDatabaseRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *databaseRepository {
	return &databaseRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}
