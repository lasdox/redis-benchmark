package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	masterName := os.Getenv("SENTINEL_MASTER_NAME")
	sentinelAddrs := os.Getenv("SENTINEL_ADDRS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	if sentinelAddrs == "" {
		sentinelAddrs = ""
	}

	sentinelAddrsSlice := strings.Split(sentinelAddrs, ",")

	if masterName == "" {
		masterName = "harness-redis"
	}

	rdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: sentinelAddrsSlice,
		Password:      redisPassword,
	})

	ctx := context.Background()

	for {
		rdb.EvalSha()

		time.Sleep(time.Minute * 10)
	}
}
