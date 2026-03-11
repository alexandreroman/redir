package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestParseOGTags(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Example Title" />
<meta property="og:description" content="Example Description" />
<meta property="og:image" content="https://example.com/image.png" />
<meta property="article:author" content="Jane Doe" />
<meta property="article:published_time" content="2025-06-15T10:00:00Z" />
</head>
<body></body>
</html>`

	og, err := ParseOGTags(strings.NewReader(html))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if og.Title != "Example Title" {
		t.Errorf("expected title %q, got %q", "Example Title", og.Title)
	}
	if og.Description != "Example Description" {
		t.Errorf("expected description %q, got %q", "Example Description", og.Description)
	}
	if og.Image != "https://example.com/image.png" {
		t.Errorf("expected image %q, got %q", "https://example.com/image.png", og.Image)
	}
	if og.Author != "Jane Doe" {
		t.Errorf("expected author %q, got %q", "Jane Doe", og.Author)
	}
	if og.PublishedTime != "2025-06-15T10:00:00Z" {
		t.Errorf("expected published_time %q, got %q", "2025-06-15T10:00:00Z", og.PublishedTime)
	}
}

func TestParseOGTags_NoTags(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>No OG</title></head><body></body></html>`
	og, err := ParseOGTags(strings.NewReader(html))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !og.Empty() {
		t.Errorf("expected empty OG tags, got %+v", og)
	}
}

func TestIsSocialBot(t *testing.T) {
	tests := []struct {
		ua   string
		want bool
	}{
		{"LinkedInBot/1.0", true},
		{"facebookexternalhit/1.1", true},
		{"Twitterbot/1.0", true},
		{"Slackbot-LinkExpanding 1.0", true},
		{"WhatsApp/2.21.4.22", true},
		{"Mozilla/5.0 (compatible; Discordbot/2.0)", true},
		{"TelegramBot (like TwitterBot)", true},
		{"Mozilla/5.0 (Windows NT 10.0) Chrome/91.0", false},
		{"curl/7.68.0", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isSocialBot(tt.ua); got != tt.want {
			t.Errorf("isSocialBot(%q) = %v, want %v", tt.ua, got, tt.want)
		}
	}
}

func TestRedirect_SocialBotGetsOGPage(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	routes := map[string]string{
		"gh": "https://github.com",
	}
	og := map[string]OGTags{
		"gh": {
			Title:       "GitHub",
			Description: "Where the world builds software",
			Image:       "https://github.githubassets.com/images/modules/open_graph/github-octocat.png",
		},
	}
	srv := NewServer(routes, rdb, og)

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	req.Header.Set("User-Agent", "LinkedInBot/1.0")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for social bot, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `og:title`) {
		t.Error("expected og:title in response body")
	}
	if !strings.Contains(body, `og:description`) {
		t.Error("expected og:description in response body")
	}
	if !strings.Contains(body, `og:image`) {
		t.Error("expected og:image in response body")
	}
	if strings.Contains(body, `meta http-equiv="refresh"`) {
		t.Error("social bot page must not contain a meta refresh redirect")
	}

	// Social bots must not increment the click counter.
	if clicks, err := mr.Get("redir:clicks:gh"); err == nil {
		t.Errorf("expected no clicks for social bot, got %s", clicks)
	}
}

func TestRedirect_RegularBrowserGets302(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	routes := map[string]string{
		"gh": "https://github.com",
	}
	og := map[string]OGTags{
		"gh": {Title: "GitHub"},
	}
	srv := NewServer(routes, rdb, og)

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0) Chrome/91.0")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 for regular browser, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://github.com" {
		t.Errorf("expected redirect to https://github.com, got %q", loc)
	}
}

func TestRedirect_NoOGTagsFallsBackTo302(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	routes := map[string]string{
		"gh": "https://github.com",
	}
	srv := NewServer(routes, rdb, nil) // no OG tags

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	req.Header.Set("User-Agent", "LinkedInBot/1.0")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 fallback when no OG tags, got %d", w.Code)
	}
}

func TestRenderOGPage(t *testing.T) {
	og := OGTags{
		Title:         "Test <Title>",
		Description:   "A \"description\"",
		Image:         "https://example.com/img.png",
		Author:        "Jane <Doe>",
		PublishedTime: "2025-06-15T10:00:00Z",
	}
	page := renderOGPage(og)

	if !strings.Contains(page, "Test &lt;Title&gt;") {
		t.Error("expected HTML-escaped title")
	}
	if !strings.Contains(page, "A &#34;description&#34;") {
		t.Error("expected HTML-escaped description")
	}
	if !strings.Contains(page, "Jane &lt;Doe&gt;") {
		t.Error("expected HTML-escaped author")
	}
	if !strings.Contains(page, `article:published_time`) {
		t.Error("expected article:published_time in rendered page")
	}
	if strings.Contains(page, `meta http-equiv="refresh"`) {
		t.Error("OG page must not contain a meta refresh redirect")
	}
}

func TestFetchOGTags_FromTestServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><head>
			<meta property="og:title" content="Test Page" />
			<meta property="og:description" content="A test" />
			<meta property="og:image" content="https://test.com/img.png" />
			<meta property="article:author" content="John Smith" />
			<meta property="article:published_time" content="2025-01-01T12:00:00Z" />
		</head><body></body></html>`))
	}))
	defer ts.Close()

	og, err := FetchOGTags(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if og.Title != "Test Page" {
		t.Errorf("expected title %q, got %q", "Test Page", og.Title)
	}
	if og.Description != "A test" {
		t.Errorf("expected description %q, got %q", "A test", og.Description)
	}
	if og.Image != "https://test.com/img.png" {
		t.Errorf("expected image %q, got %q", "https://test.com/img.png", og.Image)
	}
	if og.Author != "John Smith" {
		t.Errorf("expected author %q, got %q", "John Smith", og.Author)
	}
	if og.PublishedTime != "2025-01-01T12:00:00Z" {
		t.Errorf("expected published_time %q, got %q", "2025-01-01T12:00:00Z", og.PublishedTime)
	}
}
