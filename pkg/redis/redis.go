package redis

import (
	"log"

	"github.com/go-redis/redis"
)

var Client *redis.Client

func Init() {
	Client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Adjust as necessary for your Redis server setup
	})

	_, err := Client.Ping().Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}
