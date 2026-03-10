package main

import (
	"fmt"
	"regexp"

	"github.com/BurntSushi/toml"
)

var validSlug = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Redirect struct {
	Slug string `toml:"slug"`
	URL  string `toml:"url"`
}

type Config struct {
	Redirects []Redirect `toml:"redirects"`
}

// LoadConfig parses the TOML file and returns a slug -> URL map for O(1) route lookup.
func LoadConfig(path string) (map[string]string, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	routes := make(map[string]string, len(cfg.Redirects))
	for _, r := range cfg.Redirects {
		if !validSlug.MatchString(r.Slug) {
			return nil, fmt.Errorf("invalid slug %q: must contain only letters, digits, hyphens or underscores", r.Slug)
		}
		routes[r.Slug] = r.URL
	}
	return routes, nil
}
