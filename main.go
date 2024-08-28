package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const cacheScript = `
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
	masterName := os.Getenv("REDIS_MASTER_NAME")
	sentinelAddrs := os.Getenv("SENTINEL_ADDRS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	useSentinelStr := os.Getenv("REDIS_USE_SENTINEL")
	intervalTimeMillisStr := os.Getenv("INTERVAL_TIME_MILLIS")

	if intervalTimeMillisStr == "" {
		intervalTimeMillisStr = string((time.Minute * 10).Milliseconds())
	}

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

	rdb.Set(ctx, "pmsEventsCacheDeleteAfterTest", map[string]interface{}{}, 0)
	rdb.Set(ctx, "pmsEventsExpireDeleteAfterTest", map[string]float64{}, 0)

	sha1, err := rdb.ScriptLoad(ctx, cacheScript).Result()
	if err != nil {
		log.Fatal(err)
	}

	millisDuration, err := time.ParseDuration(fmt.Sprintf("%sms", intervalTimeMillisStr))
	if err != nil {
		log.Fatal("failed to parse millis duration: ", err)
	}
	for {
		randomKey, _ := uuid.NewUUID()
		keys := []string{"pmsEventsCacheDeleteAfterTest", "pmsEventsExpireDeleteAfterTest", "channelKey"}
		argv := []interface{}{"30000", randomKey.String(), "1", time.Now().UnixMilli()}

		slog.Info("Issuing commands")

		t1 := time.Now()
		cmd := rdb.EvalSha(ctx, sha1, keys, argv)
		if cmd.Err() != nil {
			slog.Error("Error while doing evalsha: ", cmd.Err())
		} else {
			slog.Info("Duration for evalsha: " + time.Since(t1).String())
		}

		t2 := time.Now()
		boolCmd := rdb.SetNX(ctx, fmt.Sprintf("lock-%s", randomKey.String()), 1, time.Second)
		if boolCmd.Err() != nil {
			slog.Error("Error while acquiring lock: ", cmd.Err())
		} else {
			slog.Info("Duration for acquiring lock: " + time.Since(t2).String())
		}

		time.Sleep(millisDuration)
	}
}
