package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := os.Getenv("REDIR_CONFIG")
	if configPath == "" {
		configPath = "redirects.toml"
	}

	routes, err := LoadConfig(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "redirects", len(routes))

	rdb := NewRedisClient()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	addr := os.Getenv("REDIR_ADDR")
	if addr == "" {
		addr = ":4000"
	}

	srv := NewServer(routes, rdb)
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
