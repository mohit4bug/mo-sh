package rdb

import (
	"context"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	rdb  *redis.Client
	once sync.Once
)

func GetRedisClient() *redis.Client {
	once.Do(func() {
		rdb = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})

		ctx := context.Background()
		_, err := rdb.Ping(ctx).Result()
		if err != nil {
			log.Fatal(err)
		}
	})

	return rdb
}
