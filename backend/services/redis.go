package services

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient  *redis.Client
	ctx          = context.Background()
	RedisEnabled = false
)

func InitRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "localhost:6379"
		}
		redisURL = "redis://" + host
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("[Redis] invalid REDIS_URL '%s': %v. Caching disabled.", redisURL, err)
		return
	}

	client := redis.NewClient(opt)
	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(ctxTimeout).Err(); err != nil {
		log.Printf("[Redis] connection failed to %s: %v. Caching disabled (graceful fallback).", redisURL, err)
		return
	}

	RedisClient = client
	RedisEnabled = true
	log.Printf("[Redis] connected successfully to %s", redisURL)
}

func GetCache(key string) (string, error) {
	if !RedisEnabled || RedisClient == nil {
		return "", redis.Nil
	}
	return RedisClient.Get(ctx, key).Result()
}

func SetCache(key string, value interface{}, expiration time.Duration) {
	if !RedisEnabled || RedisClient == nil {
		return
	}
	_ = RedisClient.Set(ctx, key, value, expiration).Err()
}

func DeleteCache(keys ...string) {
	if !RedisEnabled || RedisClient == nil || len(keys) == 0 {
		return
	}
	_ = RedisClient.Del(ctx, keys...).Err()
}
