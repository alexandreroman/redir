package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestServer(t *testing.T) (*Server, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	routes := map[string]string{
		"gh":   "https://github.com",
		"docs": "https://docs.example.com",
	}
	return NewServer(routes, rdb), mr
}

func TestHealthz(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("expected OK, got %q", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("expected text/plain, got %q", ct)
	}
}

func TestLLMsTxt(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("expected text/plain; charset=utf-8, got %q", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body for llms.txt")
	}
}

func TestRedirect_ValidSlug(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://github.com" {
		t.Errorf("expected redirect to https://github.com, got %q", loc)
	}
}

func TestRedirect_UnknownSlug(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRedirect_RootPath(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for root path, got %d", w.Code)
	}
}

func TestRedirect_IncrementsClick(t *testing.T) {
	srv, mr := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}

	// The click is tracked in a fire-and-forget goroutine; poll briefly.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if val, err := mr.Get("redir:clicks:gh"); err == nil && val == "1" {
			return // success
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected redir:clicks:gh to be 1 after redirect")
}

func TestQRCode_ValidSlug(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/gh.png", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png, got %q", ct)
	}
	// PNG files start with the magic bytes \x89PNG
	body := w.Body.Bytes()
	if len(body) < 4 || string(body[1:4]) != "PNG" {
		t.Error("response does not look like a valid PNG")
	}
}

func TestQRCode_UnknownSlug(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/nope.png", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestStats(t *testing.T) {
	srv, mr := newTestServer(t)

	// Seed some click data in Redis.
	mr.Set("redir:clicks:gh", "42")
	mr.Set("redir:clicks:docs", "7")

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}

	var resp statsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Meta.Version)
	}
	if resp.Meta.Date == "" {
		t.Error("expected non-empty date")
	}
	if resp.Slugs["gh"] != 42 {
		t.Errorf("expected gh=42, got %d", resp.Slugs["gh"])
	}
	if resp.Slugs["docs"] != 7 {
		t.Errorf("expected docs=7, got %d", resp.Slugs["docs"])
	}
}

func TestStats_NoClicks(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp statsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	for slug, count := range resp.Slugs {
		if count != 0 {
			t.Errorf("expected 0 clicks for %s, got %d", slug, count)
		}
	}
}
