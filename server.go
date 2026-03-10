package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"time"

	"github.com/redis/go-redis/v9"
	qrcode "github.com/skip2/go-qrcode"
)

//go:embed llms.txt.tmpl
var llmsTmpl string

type Server struct {
	routes map[string]string
	rdb    *redis.Client
}

func NewServer(routes map[string]string, rdb *redis.Client) *Server {
	return &Server{routes: routes, rdb: rdb}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/")

	if slug == "healthz" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	if slug == "stats" {
		s.handleStats(w, r)
		return
	}

	if slug == "robots.txt" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("User-agent: *\nDisallow: /\n"))
		return
	}

	if slug == "llms.txt" {
		s.handleLLMsTxt(w, r)
		return
	}

	if qrSlug, ok := strings.CutSuffix(slug, ".png"); ok {
		s.handleQRCode(w, r, qrSlug)
		return
	}

	target, ok := s.routes[slug]
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Fire-and-forget: track the click without delaying the redirect response.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := IncrClick(ctx, s.rdb, slug); err != nil {
			slog.Error("failed to increment click", "slug", slug, "error", err)
		}
	}()

	slog.Info("redirect", "slug", slug, "target", target)
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request, slug string) {
	if _, ok := s.routes[slug]; !ok {
		http.NotFound(w, r)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	redirectURL := scheme + "://" + r.Host + "/" + slug

	png, err := qrcode.Encode(redirectURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "failed to generate QR code", http.StatusInternalServerError)
		slog.Error("failed to generate QR code", "slug", slug, "error", err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (s *Server) handleLLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(llmsTmpl))
}

type statsResponse struct {
	Meta  statsMeta          `json:"meta"`
	Slugs map[string]int64   `json:"slugs"`
}

type statsMeta struct {
	Version int    `json:"version"`
	Date    string `json:"date"`
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	slugs := make([]string, 0, len(s.routes))
	for slug := range s.routes {
		slugs = append(slugs, slug)
	}

	clicks, err := GetAllClicks(r.Context(), s.rdb, slugs)
	if err != nil {
		http.Error(w, "failed to retrieve stats", http.StatusInternalServerError)
		slog.Error("failed to get stats", "error", err)
		return
	}

	resp := statsResponse{
		Meta: statsMeta{
			Version: 1,
			Date:    time.Now().Format(time.DateOnly),
		},
		Slugs: clicks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
