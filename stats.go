package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "redir:clicks:"

func NewRedisClient() *redis.Client {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	db := 0

	return redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
}

func IncrClick(ctx context.Context, rdb *redis.Client, slug string) error {
	return rdb.Incr(ctx, redisKeyPrefix+slug).Err()
}

func GetAllClicks(ctx context.Context, rdb *redis.Client, slugs []string) (map[string]int64, error) {
	stats := make(map[string]int64, len(slugs))
	for _, slug := range slugs {
		count, err := rdb.Get(ctx, redisKeyPrefix+slug).Int64()
		if err == redis.Nil {
			count = 0 // No key yet means no clicks recorded.
		} else if err != nil {
			return nil, fmt.Errorf("redis get %s: %w", slug, err)
		}
		stats[slug] = count
	}
	return stats, nil
}
