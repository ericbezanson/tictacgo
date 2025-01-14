package redis

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

var Client *redis.Client
var ctx = context.Background()

func Init() {
	Client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Adjust as necessary for your Redis server setup
	})

	_, err := Client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}
