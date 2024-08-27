package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const luaScript = `
local expireDateScore = redis.call('zscore', KEYS[2], ARGV[2])
local exists = redis.call('hexists', KEYS[1], ARGV[2]) == 1
if expireDateScore ~= false and tonumber(expireDateScore) <= tonumber(ARGV[4]) then 
    exists = false 
end
if exists then 
    return 0 
else 
    if ARGV[1] ~= '-1' then 
        redis.call('hset', KEYS[1], ARGV[2], ARGV[3]) 
        redis.call('zadd', KEYS[2], ARGV[1], ARGV[2]) 
        local msg = struct.pack('Lc0Lc0', string.len(ARGV[2]), ARGV[2], string.len(ARGV[3]), ARGV[3])
        redis.call('publish', KEYS[3], msg) 
        return 1 
    else 
        redis.call('hset', KEYS[1], ARGV[2], ARGV[3]) 
        local msg = struct.pack('Lc0Lc0', string.len(ARGV[2]), ARGV[2], string.len(ARGV[3]), ARGV[3])
        redis.call('publish', KEYS[3], msg) 
        return 1 
    end 
end`

func main() {
	var rdb *redis.Client
	redisSingleAddr := os.Getenv("REDIS_ADDR")
	masterName := os.Getenv("SENTINEL_MASTER_NAME")
	sentinelAddrs := os.Getenv("SENTINEL_ADDRS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	useSentinelStr := os.Getenv("REDIS_USE_SENTINEL")
	useSentinel := true
	if useSentinelStr != "" {
		b, err := strconv.ParseBool(useSentinelStr)
		if err != nil {
			log.Fatal(err)
		}
		useSentinel = b
	}

	if sentinelAddrs == "" {
		sentinelAddrs = ""
	}

	sentinelAddrsSlice := strings.Split(sentinelAddrs, ",")

	if masterName == "" {
		masterName = "harness-redis"
	}

	if useSentinel {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: sentinelAddrsSlice,
			Password:      redisPassword,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     redisSingleAddr,
			Password: redisPassword,
			PoolSize: 100,
		})
	}

	ctx := context.Background()

	rdb.Set(ctx, "pmsEventsCache", map[string]interface{}{}, 0)
	rdb.Set(ctx, "pmsEventsExpire", map[string]float64{}, 0)

	for {
		keys := []string{"pmsEventsCache", "pmsEventsExpire", "channelKey"}
		argv := []interface{}{"1000", "key", "value", time.Now().UnixMilli()}

		log.Println("Issuing commands")
		t1 := time.Now()

		cmd := rdb.EvalSha(ctx, luaScript)
		log.Println("Duration for evalsha: " + time.Since(t1).String())

		time.Sleep(time.Minute * 10)
	}
}
