package main

import (
	"regexp"
	"strings"
)

// extractArticleLinks finds YouTrack article IDs from text.
// Handles both full URLs and bare IDs like PROJECT-A-1.
func extractArticleLinks(text, baseURL string) []string {
	if text == "" {
		return nil
	}

	seen := make(map[string]bool)
	var ids []string

	// Pattern 1: full URL  https://youtrack.example.com/articles/PROJECT-A-123
	if baseURL != "" {
		escaped := regexp.QuoteMeta(strings.TrimRight(baseURL, "/"))
		urlPattern := regexp.MustCompile(escaped + `/articles/([A-Za-z0-9]+-A-\d+)`)
		for _, m := range urlPattern.FindAllStringSubmatch(text, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				ids = append(ids, m[1])
			}
		}
	}

	// Pattern 2: bare article ID like PROJECT-A-1 (YouTrack article ID format)
	barePattern := regexp.MustCompile(`\b([A-Z][A-Z0-9]+-A-\d+)\b`)
	for _, m := range barePattern.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}

	return ids
}

// extractArticleID normalises a user-supplied article reference.
// If it looks like a full URL, pull out the ID segment.
// Otherwise return as-is.
func extractArticleID(input, baseURL string) string {
	input = strings.TrimSpace(input)

	// If it contains /articles/ — extract the ID after it
	if idx := strings.Index(input, "/articles/"); idx != -1 {
		rest := input[idx+len("/articles/"):]
		// strip any trailing query string or fragment
		rest = strings.Split(rest, "?")[0]
		rest = strings.Split(rest, "#")[0]
		return strings.TrimRight(rest, "/")
	}

	return input
}
