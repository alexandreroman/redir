package main

import (
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html/atom"
	htmlparser "golang.org/x/net/html"
)

// OGTags holds the Open Graph metadata extracted from a target URL.
type OGTags struct {
	Title       string
	Description string
	Image       string
}

// Empty reports whether all OG fields are blank.
func (og OGTags) Empty() bool {
	return og.Title == "" && og.Description == "" && og.Image == ""
}

// FetchOGTags fetches the given URL and extracts Open Graph meta tags.
func FetchOGTags(targetURL string) (OGTags, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return OGTags{}, err
	}
	req.Header.Set("User-Agent", "redir-ogfetcher/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return OGTags{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OGTags{}, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, targetURL)
	}

	// Limit read to 1 MB to avoid downloading huge pages.
	body := io.LimitReader(resp.Body, 1<<20)
	return ParseOGTags(body)
}

// ParseOGTags parses Open Graph meta tags from an HTML reader.
func ParseOGTags(r io.Reader) (OGTags, error) {
	doc, err := htmlparser.Parse(r)
	if err != nil {
		return OGTags{}, err
	}

	var og OGTags
	var walk func(*htmlparser.Node)
	walk = func(n *htmlparser.Node) {
		if n.Type == htmlparser.ElementNode && n.DataAtom == atom.Meta {
			var property, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "property":
					property = a.Val
				case "content":
					content = a.Val
				}
			}
			switch property {
			case "og:title":
				og.Title = content
			case "og:description":
				og.Description = content
			case "og:image":
				og.Image = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return og, nil
}

// FetchAllOGTags fetches Open Graph tags for every route in parallel.
// Errors are logged but do not prevent other routes from being fetched.
func FetchAllOGTags(routes map[string]string) map[string]OGTags {
	type result struct {
		slug string
		tags OGTags
	}

	ch := make(chan result, len(routes))
	for slug, url := range routes {
		go func(s, u string) {
			tags, err := FetchOGTags(u)
			if err != nil {
				slog.Warn("failed to fetch OG tags", "slug", s, "url", u, "error", err)
				ch <- result{slug: s}
				return
			}
			if tags.Empty() {
				slog.Debug("no OG tags found", "slug", s, "url", u)
			}
			ch <- result{slug: s, tags: tags}
		}(slug, url)
	}

	ogMap := make(map[string]OGTags, len(routes))
	for range len(routes) {
		r := <-ch
		if !r.tags.Empty() {
			ogMap[r.slug] = r.tags
		}
	}
	return ogMap
}

// isSocialBot returns true if the User-Agent belongs to a social media crawler.
func isSocialBot(ua string) bool {
	ua = strings.ToLower(ua)
	bots := []string{
		"linkedinbot",
		"facebookexternalhit",
		"facebot",
		"twitterbot",
		"slackbot",
		"whatsapp",
		"discordbot",
		"telegrambot",
		"applebot",
		"pinterestbot",
		"redditbot",
	}
	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

// renderOGRedirectPage returns an HTML page that contains Open Graph meta tags
// and a meta-refresh redirect to the target URL.
func renderOGRedirectPage(target string, og OGTags) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")

	if og.Title != "" {
		fmt.Fprintf(&b, "<meta property=\"og:title\" content=\"%s\" />\n", html.EscapeString(og.Title))
	}
	if og.Description != "" {
		fmt.Fprintf(&b, "<meta property=\"og:description\" content=\"%s\" />\n", html.EscapeString(og.Description))
	}
	if og.Image != "" {
		fmt.Fprintf(&b, "<meta property=\"og:image\" content=\"%s\" />\n", html.EscapeString(og.Image))
	}

	escaped := html.EscapeString(target)
	fmt.Fprintf(&b, "<meta http-equiv=\"refresh\" content=\"0; url=%s\" />\n", escaped)
	fmt.Fprintf(&b, "</head>\n<body>\n<p>Redirecting to <a href=\"%s\">%s</a>...</p>\n</body>\n</html>\n", escaped, escaped)

	return b.String()
}
