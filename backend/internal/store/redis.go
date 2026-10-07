package store

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
})

func ConnectRedis() {
	err := RedisClient.Ping(context.Background()).Err()

	if err != nil {
		fmt.Println("[ERROR] Connecting redis error: ", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] Connected redis database")
}
