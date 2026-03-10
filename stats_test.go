package main

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return rdb, mr
}

func TestIncrClick(t *testing.T) {
	rdb, mr := newTestRedis(t)
	ctx := context.Background()

	if err := IncrClick(ctx, rdb, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mr.CheckGet(t, "redir:clicks:test", "1")

	if err := IncrClick(ctx, rdb, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mr.CheckGet(t, "redir:clicks:test", "2")
}

func TestIncrClick_MultipleSlugs(t *testing.T) {
	rdb, mr := newTestRedis(t)
	ctx := context.Background()

	IncrClick(ctx, rdb, "a")
	IncrClick(ctx, rdb, "b")
	IncrClick(ctx, rdb, "a")

	mr.CheckGet(t, "redir:clicks:a", "2")
	mr.CheckGet(t, "redir:clicks:b", "1")
}

func TestGetAllClicks(t *testing.T) {
	rdb, mr := newTestRedis(t)
	ctx := context.Background()

	mr.Set("redir:clicks:a", "10")
	mr.Set("redir:clicks:b", "20")

	clicks, err := GetAllClicks(ctx, rdb, []string{"a", "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clicks["a"] != 10 {
		t.Errorf("expected a=10, got %d", clicks["a"])
	}
	if clicks["b"] != 20 {
		t.Errorf("expected b=20, got %d", clicks["b"])
	}
}

func TestGetAllClicks_MissingSlugs(t *testing.T) {
	rdb, _ := newTestRedis(t)
	ctx := context.Background()

	clicks, err := GetAllClicks(ctx, rdb, []string{"missing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clicks["missing"] != 0 {
		t.Errorf("expected 0 for missing slug, got %d", clicks["missing"])
	}
}

func TestGetAllClicks_EmptySlugs(t *testing.T) {
	rdb, _ := newTestRedis(t)
	ctx := context.Background()

	// Redis MGET with zero keys is an error; verify we propagate it.
	_, err := GetAllClicks(ctx, rdb, []string{})
	if err == nil {
		t.Fatal("expected error for empty slug list")
	}
}

func TestGetAllClicks_MixedExistingAndMissing(t *testing.T) {
	rdb, mr := newTestRedis(t)
	ctx := context.Background()

	mr.Set("redir:clicks:exists", "5")

	clicks, err := GetAllClicks(ctx, rdb, []string{"exists", "nope"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clicks["exists"] != 5 {
		t.Errorf("expected exists=5, got %d", clicks["exists"])
	}
	if clicks["nope"] != 0 {
		t.Errorf("expected nope=0, got %d", clicks["nope"])
	}
}
