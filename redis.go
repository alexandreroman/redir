package main

import (
	"cmp"
	"crypto/tls"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	host := cmp.Or(os.Getenv("REDIS_HOST"), "localhost")
	port := cmp.Or(os.Getenv("REDIS_PORT"), "6379")

	addr := host + ":" + port
	slog.Info("connecting to redis", "addr", addr)

	opts := &redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
	}
	if os.Getenv("REDIS_TLS") == "true" {
		opts.TLSConfig = &tls.Config{}
	}

	return redis.NewClient(opts)
}
