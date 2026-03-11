package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html/atom"
	htmlparser "golang.org/x/net/html"
)

// OGTags holds the Open Graph metadata extracted from a target URL.
type OGTags struct {
	Title         string
	Type          string
	Description   string
	Image         string
	Author        string
	PublishedTime string
}

// Empty reports whether all OG fields are blank.
func (og OGTags) Empty() bool {
	return og.Title == "" && og.Type == "" && og.Description == "" && og.Image == "" && og.Author == "" && og.PublishedTime == ""
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
			case "og:type":
				og.Type = content
			case "og:description":
				og.Description = content
			case "og:image":
				og.Image = content
			case "article:author":
				og.Author = content
			case "article:published_time":
				og.PublishedTime = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return og, nil
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

// renderOGPage returns an HTML page that contains only Open Graph meta tags.
// No redirect is performed — this page is meant for social media crawlers only.
func renderOGPage(og OGTags) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\" />\n")

	if og.Title != "" {
		escaped := html.EscapeString(og.Title)
		fmt.Fprintf(&b, "<meta name=\"title\" property=\"og:title\" content=\"%s\" />\n", escaped)
		fmt.Fprintf(&b, "<title>%s</title>\n", escaped)
	}

	ogType := og.Type
	if ogType == "" {
		ogType = "website"
	}
	fmt.Fprintf(&b, "<meta property=\"og:type\" content=\"%s\" />\n", html.EscapeString(ogType))

	if og.Description != "" {
		escaped := html.EscapeString(og.Description)
		fmt.Fprintf(&b, "<meta name=\"description\" property=\"og:description\" content=\"%s\" />\n", escaped)
	}
	if og.Image != "" {
		escaped := html.EscapeString(og.Image)
		fmt.Fprintf(&b, "<meta name=\"image\" property=\"og:image\" content=\"%s\" />\n", escaped)
	}
	if og.Author != "" {
		escaped := html.EscapeString(og.Author)
		fmt.Fprintf(&b, "<meta name=\"author\" property=\"article:author\" content=\"%s\" />\n", escaped)
	}
	if og.PublishedTime != "" {
		fmt.Fprintf(&b, "<meta property=\"article:published_time\" content=\"%s\" />\n", html.EscapeString(og.PublishedTime))
	}

	b.WriteString("</head>\n<body></body>\n</html>\n")

	return b.String()
}
