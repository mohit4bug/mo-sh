package rdb

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

func InitRedis() {
	redisOnce.Do(func() {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := redisClient.Ping(ctx).Result(); err != nil {
			log.Fatal(err)
		}
	})
}

func GetRedis() *redis.Client {
	if redisClient == nil {
		log.Panic("Redis client is not initialized")
	}
	return redisClient
}

func CloseRedis() {
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Fatal(err)
		} else {
			log.Println("Redis connection closed")
		}
	}
}
