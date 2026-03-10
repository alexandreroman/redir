package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
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

	addr := host + ":" + port
	slog.Info("connecting to redis", "addr", addr)

	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}
	if os.Getenv("REDIS_TLS") == "true" {
		opts.TLSConfig = &tls.Config{}
	}

	return redis.NewClient(opts)
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
