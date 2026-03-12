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

//go:embed index.html
var indexHTML []byte

type Server struct {
	routes map[string]string
	rdb    *redis.Client
}

func NewServer(routes map[string]string, rdb *redis.Client) *Server {
	return &Server{routes: routes, rdb: rdb}
}

// clientIP returns the client's real IP address, using the X-Forwarded-For
// header when present (first entry), falling back to r.RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
}

func notFound(w http.ResponseWriter, r *http.Request, slug string) {
	slog.Warn("not found", "slug", slug, "method", r.Method, "remote", clientIP(r))
	http.NotFound(w, r)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/")

	if slug == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
		return
	}

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
		handleRobotsTxt(w)
		return
	}

	if qrSlug, ok := strings.CutSuffix(slug, ".png"); ok {
		s.handleQRCode(w, r, qrSlug)
		return
	}

	target, ok := s.routes[slug]
	if !ok {
		notFound(w, r, slug)
		return
	}

	// Serve an HTML page with Open Graph meta tags to social media crawlers
	// so that link previews render correctly on LinkedIn, Facebook, etc.
	// Social bots do not trigger a redirect or increment the click counter.
	// OG tags are fetched on-demand when a social bot requests the page.
	if isSocialBot(r.UserAgent()) {
		og, err := FetchOGTags(target)
		if err != nil {
			slog.Warn("failed to fetch OG tags", "slug", slug, "url", target, "error", err)
		} else if !og.Empty() {
			slog.Info("social bot", "slug", slug, "target", target, "user_agent", r.UserAgent())
			page := renderOGPage(og, target)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(page))
			return
		}
	}

	// Fire-and-forget: track the click without delaying the redirect response.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := IncrClick(ctx, s.rdb, slug); err != nil {
			slog.Error("failed to increment click", "slug", slug, "error", err)
		}
	}()

	if ua := r.UserAgent(); ua != "" {
		slog.Info("redirect", "slug", slug, "target", target, "remote", clientIP(r), "user_agent", ua)
	} else {
		slog.Info("redirect", "slug", slug, "target", target, "remote", clientIP(r))
	}

	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request, slug string) {
	if _, ok := s.routes[slug]; !ok {
		notFound(w, r, slug)
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

func handleRobotsTxt(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("User-agent: *\nAllow: /\n"))
}

type statsResponse struct {
	Meta  statsMeta        `json:"meta"`
	Slugs map[string]int64 `json:"slugs"`
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
