package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "redir:clicks:"

func IncrClick(ctx context.Context, rdb *redis.Client, slug string) error {
	return rdb.Incr(ctx, redisKeyPrefix+slug).Err()
}

func GetAllClicks(ctx context.Context, rdb *redis.Client, slugs []string) (map[string]int64, error) {
	keys := make([]string, len(slugs))
	for i, slug := range slugs {
		keys[i] = redisKeyPrefix + slug
	}

	vals, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget: %w", err)
	}

	stats := make(map[string]int64, len(slugs))
	for i, slug := range slugs {
		if vals[i] == nil {
			stats[slug] = 0
			continue
		}
		s, ok := vals[i].(string)
		if !ok {
			return nil, fmt.Errorf("redis mget: unexpected type for %s", slug)
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("redis mget: invalid value for %s: %w", slug, err)
		}
		stats[slug] = n
	}
	return stats, nil
}
