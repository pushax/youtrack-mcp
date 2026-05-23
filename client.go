package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type YouTrackClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewYouTrackClient(baseURL, token string) *YouTrackClient {
	return &YouTrackClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *YouTrackClient) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + "/api" + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ---- Issue ----

type Issue struct {
	ID          string
	Summary     string
	Description string
	State       string
	Assignee    string
	Reporter    string
	Created     time.Time
	Updated     time.Time
	URL         string
}

type ytIssue struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Fields      []struct {
		Name  string `json:"name"`
		Value any    `json:"value"`
	} `json:"customFields"`
}

func (c *YouTrackClient) GetIssue(ctx context.Context, issueID string) (*Issue, error) {
	params := url.Values{}
	params.Set("fields", "id,summary,description,customFields(name,value(name,login))")

	body, err := c.get(ctx, "/issues/"+url.PathEscape(issueID), params)
	if err != nil {
		return nil, err
	}

	var raw ytIssue
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	issue := &Issue{
		ID:          raw.ID,
		Summary:     raw.Summary,
		Description: raw.Description,
		URL:         fmt.Sprintf("%s/issue/%s", c.baseURL, raw.ID),
	}

	// Extract state and assignee from custom fields
	for _, f := range raw.Fields {
		switch f.Name {
		case "State":
			if v, ok := f.Value.(map[string]any); ok {
				if name, ok := v["name"].(string); ok {
					issue.State = name
				}
			}
		case "Assignee":
			if v, ok := f.Value.(map[string]any); ok {
				if name, ok := v["name"].(string); ok {
					issue.Assignee = name
				}
			}
		case "reporter", "Reporter":
			if v, ok := f.Value.(map[string]any); ok {
				if name, ok := v["name"].(string); ok {
					issue.Reporter = name
				}
			}
		}
	}

	return issue, nil
}

func (issue *Issue) Format() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Issue %s\n\n", issue.ID))
	sb.WriteString(fmt.Sprintf("**Summary:** %s\n", issue.Summary))
	if issue.State != "" {
		sb.WriteString(fmt.Sprintf("**State:** %s\n", issue.State))
	}
	if issue.Assignee != "" {
		sb.WriteString(fmt.Sprintf("**Assignee:** %s\n", issue.Assignee))
	}
	sb.WriteString(fmt.Sprintf("**URL:** %s\n", issue.URL))

	if issue.Description != "" {
		sb.WriteString("\n## Description\n\n")
		sb.WriteString(issue.Description)
	}

	return sb.String()
}

// ---- Search ----

type IssueSummary struct {
	ID      string
	Summary string
	State   string
}

func (c *YouTrackClient) SearchIssues(ctx context.Context, query string, limit int) ([]IssueSummary, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("fields", "id,summary,customFields(name,value(name))")
	params.Set("$top", fmt.Sprintf("%d", limit))

	body, err := c.get(ctx, "/issues", params)
	if err != nil {
		return nil, err
	}

	var raw []ytIssue
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	results := make([]IssueSummary, 0, len(raw))
	for _, r := range raw {
		s := IssueSummary{
			ID:      r.ID,
			Summary: r.Summary,
		}
		for _, f := range r.Fields {
			if f.Name == "State" {
				if v, ok := f.Value.(map[string]any); ok {
					if name, ok := v["name"].(string); ok {
						s.State = name
					}
				}
			}
		}
		results = append(results, s)
	}

	return results, nil
}

// ---- Article ----

type Article struct {
	ID      string
	Title   string
	Content string
	URL     string
}

type ytArticle struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

func (c *YouTrackClient) GetArticle(ctx context.Context, articleID string) (*Article, error) {
	params := url.Values{}
	params.Set("fields", "id,summary,content")

	body, err := c.get(ctx, "/articles/"+url.PathEscape(articleID), params)
	if err != nil {
		return nil, err
	}

	var raw ytArticle
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &Article{
		ID:      raw.ID,
		Title:   raw.Summary,
		Content: raw.Content,
		URL:     fmt.Sprintf("%s/articles/%s", c.baseURL, raw.ID),
	}, nil
}

func (a *Article) Format() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Article: %s\n", a.Title))
	sb.WriteString(fmt.Sprintf("_ID: %s | URL: %s_\n\n", a.ID, a.URL))
	if a.Content != "" {
		sb.WriteString(a.Content)
	} else {
		sb.WriteString("_(no content)_")
	}
	return sb.String()
}

// ---- Comments ----

type Comment struct {
	ID      string
	Text    string
	Author  string
	Created time.Time
}

type ytComment struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Author  struct {
		Name string `json:"name"`
	} `json:"author"`
	Created int64 `json:"created"`
}

func (c *YouTrackClient) GetIssueComments(ctx context.Context, issueID string) ([]Comment, error) {
	params := url.Values{}
	params.Set("fields", "id,text,author(name),created")

	body, err := c.get(ctx, "/issues/"+url.PathEscape(issueID)+"/comments", params)
	if err != nil {
		return nil, err
	}

	var raw []ytComment
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	comments := make([]Comment, 0, len(raw))
	for _, r := range raw {
		comments = append(comments, Comment{
			ID:      r.ID,
			Text:    r.Text,
			Author:  r.Author.Name,
			Created: time.UnixMilli(r.Created),
		})
	}
	return comments, nil
}

func (c *YouTrackClient) AddComment(ctx context.Context, issueID, text string) error {
	payload := map[string]string{"text": text}
	_, err := c.post(ctx, "/issues/"+url.PathEscape(issueID)+"/comments", payload)
	return err
}

// ---- Create / Update Issue ----

func (c *YouTrackClient) CreateIssue(ctx context.Context, projectID, summary, description string) (*Issue, error) {
	payload := map[string]any{
		"summary":     summary,
		"description": description,
		"project":     map[string]string{"id": projectID},
	}

	body, err := c.post(ctx, "/issues", payload)
	if err != nil {
		return nil, err
	}

	var raw ytIssue
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &Issue{
		ID:          raw.ID,
		Summary:     raw.Summary,
		Description: raw.Description,
		URL:         fmt.Sprintf("%s/issue/%s", c.baseURL, raw.ID),
	}, nil
}

// UpdateIssue sets arbitrary fields via the command API (simplest way to change state/assignee/etc.)
func (c *YouTrackClient) UpdateIssue(ctx context.Context, issueID, command string) error {
	payload := map[string]any{
		"query": command,
		"issues": []map[string]string{
			{"id": issueID},
		},
	}
	_, err := c.post(ctx, "/commands", payload)
	return err
}

// ---- List Project Issues ----

func (c *YouTrackClient) ListProjectIssues(ctx context.Context, projectID string, limit int) ([]IssueSummary, error) {
	return c.SearchIssues(ctx, "project: "+projectID, limit)
}

// ---- List / Search Articles ----

type ArticleSummary struct {
	ID       string
	Title    string
	ParentID string
	URL      string
}

type ytArticleSummary struct {
	ID            string `json:"id"`
	Summary       string `json:"summary"`
	ParentArticle *struct {
		ID string `json:"id"`
	} `json:"parentArticle"`
}

func (c *YouTrackClient) fetchArticles(ctx context.Context, params url.Values) ([]ArticleSummary, error) {
	params.Set("fields", "id,summary,parentArticle(id)")

	body, err := c.get(ctx, "/articles", params)
	if err != nil {
		return nil, err
	}

	var raw []ytArticleSummary
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	result := make([]ArticleSummary, 0, len(raw))
	for _, r := range raw {
		a := ArticleSummary{
			ID:    r.ID,
			Title: r.Summary,
			URL:   fmt.Sprintf("%s/articles/%s", c.baseURL, r.ID),
		}
		if r.ParentArticle != nil {
			a.ParentID = r.ParentArticle.ID
		}
		result = append(result, a)
	}
	return result, nil
}

func (c *YouTrackClient) ListArticles(ctx context.Context, projectID string) ([]ArticleSummary, error) {
	params := url.Values{}
	if projectID != "" {
		params.Set("query", "project: "+projectID)
	}
	params.Set("$top", "100")
	return c.fetchArticles(ctx, params)
}

func (c *YouTrackClient) SearchArticles(ctx context.Context, query string) ([]ArticleSummary, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("$top", "20")
	return c.fetchArticles(ctx, params)
}

func (c *YouTrackClient) CreateArticle(ctx context.Context, projectID, title, content string) (*Article, error) {
	payload := map[string]any{
		"summary": title,
		"content": content,
		"project": map[string]string{"id": projectID},
	}

	body, err := c.post(ctx, "/articles", payload)
	if err != nil {
		return nil, err
	}

	var raw ytArticle
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &Article{
		ID:      raw.ID,
		Title:   raw.Summary,
		Content: raw.Content,
		URL:     fmt.Sprintf("%s/articles/%s", c.baseURL, raw.ID),
	}, nil
}

// ---- Merge Requests (VCS) ----

type MergeRequest struct {
	ID      string
	Title   string
	URL     string
	Author  string
	State   string
	Created time.Time
}

type ytVcsChange struct {
	Type   string `json:"$type"`
	ID     string `json:"id"`
	Text   string `json:"text"`
	URL    string `json:"url"`
	Author struct {
		Name  string `json:"name"`
		Login string `json:"login"`
	} `json:"author"`
	Date  int64  `json:"date"`
	State string `json:"state"`
}

func (c *YouTrackClient) GetIssueMergeRequests(ctx context.Context, issueID string) ([]MergeRequest, error) {
	params := url.Values{}
	params.Set("fields", "$type,id,text,url,author(name,login),date,state")

	body, err := c.get(ctx, "/issues/"+url.PathEscape(issueID)+"/vcsChanges", params)
	if err != nil {
		return nil, err
	}

	var raw []ytVcsChange
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	var mrs []MergeRequest
	for _, r := range raw {
		typeLower := strings.ToLower(r.Type)
		if !strings.Contains(typeLower, "pullrequest") && !strings.Contains(typeLower, "mergerequest") {
			continue
		}

		author := r.Author.Name
		if author == "" {
			author = r.Author.Login
		}

		mrs = append(mrs, MergeRequest{
			ID:      r.ID,
			Title:   r.Text,
			URL:     r.URL,
			Author:  author,
			State:   r.State,
			Created: time.UnixMilli(r.Date),
		})
	}

	return mrs, nil
}

// ---- HTTP helpers ----

func (c *YouTrackClient) post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/api"+path+"?fields=id,summary,content,description",
		strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
