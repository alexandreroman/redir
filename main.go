package main

import (
	"cmp"
	"context"
	"log/slog"
	"net/http"
	"os"
)

var gitCommit = "unknown"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	slog.Info("starting redir", "commit", gitCommit)

	configPath := cmp.Or(os.Getenv("REDIR_CONFIG"), "redirects.toml")

	routes, err := LoadConfig(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "redirects", len(routes))

	rdb := NewRedisClient()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("redis is not available, stats tracking will be disabled until reconnection", "error", err)
	}

	addr := cmp.Or(os.Getenv("REDIR_ADDR"), ":4000")

	srv := NewServer(routes, rdb)
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
