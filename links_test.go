package main

import (
	"testing"
)

func TestExtractArticleLinks(t *testing.T) {
	const base = "https://youtrack.example.com"

	tests := []struct {
		name    string
		text    string
		baseURL string
		want    []string
	}{
		{
			name:    "full URL",
			text:    "See https://youtrack.example.com/articles/CS-A-42 for details",
			baseURL: base,
			want:    []string{"CS-A-42"},
		},
		{
			name:    "bare ID",
			text:    "Related: CS-A-42",
			baseURL: base,
			want:    []string{"CS-A-42"},
		},
		{
			name:    "full URL and bare ID deduplicated",
			text:    "https://youtrack.example.com/articles/CS-A-42 and CS-A-42",
			baseURL: base,
			want:    []string{"CS-A-42"},
		},
		{
			name:    "multiple unique articles",
			text:    "CS-A-1 and CS-A-2",
			baseURL: base,
			want:    []string{"CS-A-1", "CS-A-2"},
		},
		{
			name:    "multiple full URLs",
			text:    "https://youtrack.example.com/articles/CS-A-1 https://youtrack.example.com/articles/CS-A-2",
			baseURL: base,
			want:    []string{"CS-A-1", "CS-A-2"},
		},
		{
			name:    "issue ID not matched (no -A-)",
			text:    "CS-123 is not an article",
			baseURL: base,
			want:    nil,
		},
		{
			name:    "empty text",
			text:    "",
			baseURL: base,
			want:    nil,
		},
		{
			name:    "no articles",
			text:    "nothing relevant here",
			baseURL: base,
			want:    nil,
		},
		{
			name:    "base URL with trailing slash",
			text:    "https://youtrack.example.com/articles/CS-A-7",
			baseURL: base + "/",
			want:    []string{"CS-A-7"},
		},
		{
			name:    "empty baseURL falls back to bare ID",
			text:    "CS-A-99",
			baseURL: "",
			want:    []string{"CS-A-99"},
		},
		{
			name:    "lowercase project not matched by bare ID pattern",
			text:    "cs-A-1",
			baseURL: base,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractArticleLinks(tt.text, tt.baseURL)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractArticleID(t *testing.T) {
	const base = "https://youtrack.example.com"

	tests := []struct {
		name    string
		input   string
		baseURL string
		want    string
	}{
		{
			name:    "full URL",
			input:   "https://youtrack.example.com/articles/CS-A-42",
			baseURL: base,
			want:    "CS-A-42",
		},
		{
			name:    "full URL with trailing slash",
			input:   "https://youtrack.example.com/articles/CS-A-42/",
			baseURL: base,
			want:    "CS-A-42",
		},
		{
			name:    "full URL with query string",
			input:   "https://youtrack.example.com/articles/CS-A-42?tab=content",
			baseURL: base,
			want:    "CS-A-42",
		},
		{
			name:    "full URL with fragment",
			input:   "https://youtrack.example.com/articles/CS-A-42#section",
			baseURL: base,
			want:    "CS-A-42",
		},
		{
			name:    "bare ID returned as-is",
			input:   "CS-A-42",
			baseURL: base,
			want:    "CS-A-42",
		},
		{
			name:    "whitespace trimmed",
			input:   "  CS-A-42  ",
			baseURL: base,
			want:    "CS-A-42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractArticleID(tt.input, tt.baseURL)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
