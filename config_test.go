package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "redirects.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_Valid(t *testing.T) {
	path := writeTempTOML(t, `
[[redirects]]
slug = "gh"
url  = "https://github.com"

[[redirects]]
slug = "my-site_1"
url  = "https://example.com"
`)

	routes, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes["gh"] != "https://github.com" {
		t.Errorf("unexpected URL for gh: %s", routes["gh"])
	}
	if routes["my-site_1"] != "https://example.com" {
		t.Errorf("unexpected URL for my-site_1: %s", routes["my-site_1"])
	}
}

func TestLoadConfig_Empty(t *testing.T) {
	path := writeTempTOML(t, "")

	routes, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Fatalf("expected 0 routes, got %d", len(routes))
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/redirects.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	path := writeTempTOML(t, `not valid toml [[[`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestLoadConfig_InvalidSlug(t *testing.T) {
	cases := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"spaces", "has space"},
		{"special chars", "hello!world"},
		{"slash", "a/b"},
		{"dot", "a.b"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempTOML(t, `
[[redirects]]
slug = "`+tc.slug+`"
url  = "https://example.com"
`)
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected error for invalid slug %q", tc.slug)
			}
		})
	}
}

func TestLoadConfig_InvalidURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<h1>hi</h1>"},
		{"ftp scheme", "ftp://example.com/file"},
		{"no scheme", "example.com"},
		{"empty url", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempTOML(t, `
[[redirects]]
slug = "test"
url  = "`+tc.url+`"
`)
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected error for invalid URL %q", tc.url)
			}
		})
	}
}

func TestLoadConfig_ReservedSlug(t *testing.T) {
	reserved := []string{"healthz", "stats", "robots.txt"}

	for _, slug := range reserved {
		t.Run(slug, func(t *testing.T) {
			path := writeTempTOML(t, `
[[redirects]]
slug = "`+slug+`"
url  = "https://example.com"
`)
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected error for reserved slug %q", slug)
			}
		})
	}
}
