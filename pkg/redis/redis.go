package redis

import (
	"context"
	"encoding/json"
	"fmt"
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

// UpdateLobby updates the "Conns" property of all entries in Redis to an empty array
func UpdateLobby() error {
	fmt.Println("Updating REDIS:")

	cursor := uint64(0)
	const batchSize = 100
	for {
		var keys []string
		var err error
		keys, cursor, err = Client.Scan(ctx, cursor, "", batchSize).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			// Get the value for the key
			val, err := Client.Get(ctx, key).Result()
			if err != nil {
				// handle error here, e.g. log the error and continue
				log.Printf("error getting key %s: %v", key, err)
				continue
			}

			// Unmarshal the value into a map
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(val), &data); err != nil {
				// handle error here, e.g. log the error and continue
				log.Printf("error unmarshalling key %s: %v", key, err)
				continue
			}

			// Update the "Conns" property to an empty array
			data["Conns"] = []interface{}{}

			// Marshal the updated data back to JSON
			updatedVal, err := json.Marshal(data)
			if err != nil {
				// handle error here, e.g. log the error and continue
				log.Printf("error marshalling key %s: %v", key, err)
				continue
			}

			// Set the updated value back to Redis
			if err := Client.Set(ctx, key, updatedVal, 0).Err(); err != nil {
				// handle error here, e.g. log the error and continue
				log.Printf("error setting key %s: %v", key, err)
				continue
			}
		}

		// If cursor is 0, we've processed all keys
		if cursor == 0 {
			break
		}
	}

	return nil
}
