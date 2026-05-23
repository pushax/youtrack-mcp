package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	baseURL := os.Getenv("YOUTRACK_URL")
	token := os.Getenv("YOUTRACK_TOKEN")

	if baseURL == "" || token == "" {
		log.Fatal("YOUTRACK_URL and YOUTRACK_TOKEN must be set")
	}

	client := NewYouTrackClient(baseURL, token)

	s := server.NewMCPServer(
		"youtrack-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Tool: get_issue
	s.AddTool(
		mcp.NewTool("get_issue",
			mcp.WithDescription("Get a YouTrack issue by ID. Returns summary, description, state, assignee, and any linked article URLs found in the description or custom fields."),
			mcp.WithString("issue_id",
				mcp.Required(),
				mcp.Description("Issue ID, e.g. PROJECT-123"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			issueID, ok := req.Params.Arguments["issue_id"].(string)
			if !ok || issueID == "" {
				return mcp.NewToolResultError("issue_id is required"), nil
			}

			issue, err := client.GetIssue(ctx, issueID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get issue: %v", err)), nil
			}

			return mcp.NewToolResultText(issue.Format()), nil
		},
	)

	// Tool: get_article
	s.AddTool(
		mcp.NewTool("get_article",
			mcp.WithDescription("Get a YouTrack Knowledge Base article by its ID or URL. Returns the full article content in markdown."),
			mcp.WithString("article_id",
				mcp.Required(),
				mcp.Description("Article ID (e.g. PROJECT-A-1) or full YouTrack article URL"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			articleID, ok := req.Params.Arguments["article_id"].(string)
			if !ok || articleID == "" {
				return mcp.NewToolResultError("article_id is required"), nil
			}

			articleID = extractArticleID(articleID, baseURL)

			article, err := client.GetArticle(ctx, articleID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get article: %v", err)), nil
			}

			return mcp.NewToolResultText(article.Format()), nil
		},
	)

	// Tool: get_issue_with_docs
	// Convenience tool: fetches issue + all linked articles in one shot
	s.AddTool(
		mcp.NewTool("get_issue_with_docs",
			mcp.WithDescription("Get a YouTrack issue and automatically fetch all linked Knowledge Base articles found in its description. This is the main tool for understanding a task with its documentation context."),
			mcp.WithString("issue_id",
				mcp.Required(),
				mcp.Description("Issue ID, e.g. PROJECT-123"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			issueID, ok := req.Params.Arguments["issue_id"].(string)
			if !ok || issueID == "" {
				return mcp.NewToolResultError("issue_id is required"), nil
			}

			issue, err := client.GetIssue(ctx, issueID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get issue: %v", err)), nil
			}

			result := issue.Format()

			articleIDs := extractArticleLinks(issue.Description, baseURL)
			if len(articleIDs) == 0 {
				result += "\n\n---\n_No linked Knowledge Base articles found in description._"
				return mcp.NewToolResultText(result), nil
			}

			result += fmt.Sprintf("\n\n---\n## Linked Articles (%d)\n", len(articleIDs))

			for _, aid := range articleIDs {
				article, err := client.GetArticle(ctx, aid)
				if err != nil {
					result += fmt.Sprintf("\n### ⚠️ Failed to load article %s\n%v\n", aid, err)
					continue
				}
				result += "\n" + article.Format() + "\n---\n"
			}

			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: search_issues
	s.AddTool(
		mcp.NewTool("search_issues",
			mcp.WithDescription("Search YouTrack issues using YouTrack query language. Returns list of matching issues with ID, summary, and state."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("YouTrack search query, e.g. 'project: MYPROJECT assignee: me state: Open'"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results to return (default: 20)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, ok := req.Params.Arguments["query"].(string)
			if !ok || query == "" {
				return mcp.NewToolResultError("query is required"), nil
			}

			limit := 20
			if l, ok := req.Params.Arguments["limit"].(float64); ok && l > 0 {
				limit = int(l)
			}

			issues, err := client.SearchIssues(ctx, query, limit)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
			}

			if len(issues) == 0 {
				return mcp.NewToolResultText("No issues found matching the query."), nil
			}

			result := fmt.Sprintf("Found %d issue(s):\n\n", len(issues))
			for _, issue := range issues {
				result += fmt.Sprintf("- **%s** [%s] %s\n", issue.ID, issue.State, issue.Summary)
			}

			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: get_issue_comments
	s.AddTool(
		mcp.NewTool("get_issue_comments",
			mcp.WithDescription("Get all comments for a YouTrack issue."),
			mcp.WithString("issue_id",
				mcp.Required(),
				mcp.Description("Issue ID, e.g. PROJECT-123"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			issueID, ok := req.Params.Arguments["issue_id"].(string)
			if !ok || issueID == "" {
				return mcp.NewToolResultError("issue_id is required"), nil
			}

			comments, err := client.GetIssueComments(ctx, issueID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get comments: %v", err)), nil
			}

			if len(comments) == 0 {
				return mcp.NewToolResultText("No comments found."), nil
			}

			result := fmt.Sprintf("## Comments for %s (%d)\n\n", issueID, len(comments))
			for _, c := range comments {
				result += fmt.Sprintf("**%s** _%s_\n%s\n\n---\n",
					c.Author, c.Created.Format("2006-01-02 15:04"), c.Text)
			}
			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: add_comment
	s.AddTool(
		mcp.NewTool("add_comment",
			mcp.WithDescription("Add a comment to a YouTrack issue."),
			mcp.WithString("issue_id",
				mcp.Required(),
				mcp.Description("Issue ID, e.g. PROJECT-123"),
			),
			mcp.WithString("text",
				mcp.Required(),
				mcp.Description("Comment text (markdown supported)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			issueID, _ := req.Params.Arguments["issue_id"].(string)
			text, _ := req.Params.Arguments["text"].(string)
			if issueID == "" || text == "" {
				return mcp.NewToolResultError("issue_id and text are required"), nil
			}

			if err := client.AddComment(ctx, issueID, text); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to add comment: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Comment added to %s.", issueID)), nil
		},
	)

	// Tool: update_issue
	s.AddTool(
		mcp.NewTool("update_issue",
			mcp.WithDescription("Update a YouTrack issue using a command string. Commands follow YouTrack command syntax, e.g. 'state In Progress', 'assignee john.doe', 'priority Critical'."),
			mcp.WithString("issue_id",
				mcp.Required(),
				mcp.Description("Issue ID, e.g. PROJECT-123"),
			),
			mcp.WithString("command",
				mcp.Required(),
				mcp.Description("YouTrack command, e.g. 'state In Progress' or 'assignee john.doe priority Critical'"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			issueID, _ := req.Params.Arguments["issue_id"].(string)
			command, _ := req.Params.Arguments["command"].(string)
			if issueID == "" || command == "" {
				return mcp.NewToolResultError("issue_id and command are required"), nil
			}

			if err := client.UpdateIssue(ctx, issueID, command); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to update issue: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Issue %s updated: %s", issueID, command)), nil
		},
	)

	// Tool: create_issue
	s.AddTool(
		mcp.NewTool("create_issue",
			mcp.WithDescription("Create a new YouTrack issue in a project."),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project short name (key), e.g. CS, BACKEND"),
			),
			mcp.WithString("summary",
				mcp.Required(),
				mcp.Description("Issue title/summary"),
			),
			mcp.WithString("description",
				mcp.Description("Issue description (markdown supported)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, _ := req.Params.Arguments["project_id"].(string)
			summary, _ := req.Params.Arguments["summary"].(string)
			description, _ := req.Params.Arguments["description"].(string)

			if projectID == "" || summary == "" {
				return mcp.NewToolResultError("project_id and summary are required"), nil
			}

			issue, err := client.CreateIssue(ctx, projectID, summary, description)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create issue: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Created issue %s: %s\n%s", issue.ID, issue.Summary, issue.URL)), nil
		},
	)

	// Tool: list_project_issues
	s.AddTool(
		mcp.NewTool("list_project_issues",
			mcp.WithDescription("List issues in a YouTrack project."),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project short name (key), e.g. CS, BACKEND"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results (default: 20)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, _ := req.Params.Arguments["project_id"].(string)
			if projectID == "" {
				return mcp.NewToolResultError("project_id is required"), nil
			}

			limit := 20
			if l, ok := req.Params.Arguments["limit"].(float64); ok && l > 0 {
				limit = int(l)
			}

			issues, err := client.ListProjectIssues(ctx, projectID, limit)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list issues: %v", err)), nil
			}

			if len(issues) == 0 {
				return mcp.NewToolResultText("No issues found."), nil
			}

			result := fmt.Sprintf("## Issues in %s (%d)\n\n", projectID, len(issues))
			for _, issue := range issues {
				result += fmt.Sprintf("- **%s** [%s] %s\n", issue.ID, issue.State, issue.Summary)
			}
			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: list_articles
	s.AddTool(
		mcp.NewTool("list_articles",
			mcp.WithDescription("List Knowledge Base articles in a YouTrack project. Returns article tree with IDs and titles."),
			mcp.WithString("project_id",
				mcp.Description("Project short name to filter articles (optional)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, _ := req.Params.Arguments["project_id"].(string)

			articles, err := client.ListArticles(ctx, projectID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list articles: %v", err)), nil
			}

			if len(articles) == 0 {
				return mcp.NewToolResultText("No articles found."), nil
			}

			result := fmt.Sprintf("## Articles (%d)\n\n", len(articles))
			for _, a := range articles {
				prefix := "  "
				if a.ParentID == "" {
					prefix = "▸ "
				}
				result += fmt.Sprintf("%s**%s** %s\n  %s\n", prefix, a.ID, a.Title, a.URL)
			}
			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: search_articles
	s.AddTool(
		mcp.NewTool("search_articles",
			mcp.WithDescription("Search YouTrack Knowledge Base articles by text query."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query text"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, _ := req.Params.Arguments["query"].(string)
			if query == "" {
				return mcp.NewToolResultError("query is required"), nil
			}

			articles, err := client.SearchArticles(ctx, query)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to search articles: %v", err)), nil
			}

			if len(articles) == 0 {
				return mcp.NewToolResultText("No articles found."), nil
			}

			result := fmt.Sprintf("Found %d article(s):\n\n", len(articles))
			for _, a := range articles {
				result += fmt.Sprintf("- **%s** %s\n  %s\n", a.ID, a.Title, a.URL)
			}
			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: get_issue_mrs
	s.AddTool(
		mcp.NewTool("get_issue_mrs",
			mcp.WithDescription("Get all merge requests (pull requests) linked to a YouTrack issue via VCS integration."),
			mcp.WithString("issue_id",
				mcp.Required(),
				mcp.Description("Issue ID, e.g. PROJECT-123"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			issueID, ok := req.Params.Arguments["issue_id"].(string)
			if !ok || issueID == "" {
				return mcp.NewToolResultError("issue_id is required"), nil
			}

			mrs, err := client.GetIssueMergeRequests(ctx, issueID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get merge requests: %v", err)), nil
			}

			if len(mrs) == 0 {
				return mcp.NewToolResultText(fmt.Sprintf("No merge requests linked to %s.", issueID)), nil
			}

			result := fmt.Sprintf("## Merge Requests for %s (%d)\n\n", issueID, len(mrs))
			for _, mr := range mrs {
				result += fmt.Sprintf("### %s\n", mr.Title)
				if mr.State != "" {
					result += fmt.Sprintf("**State:** %s\n", mr.State)
				}
				if mr.Author != "" {
					result += fmt.Sprintf("**Author:** %s\n", mr.Author)
				}
				if !mr.Created.IsZero() {
					result += fmt.Sprintf("**Created:** %s\n", mr.Created.Format("2006-01-02 15:04"))
				}
				if mr.URL != "" {
					result += fmt.Sprintf("**URL:** %s\n", mr.URL)
				}
				result += "\n"
			}
			return mcp.NewToolResultText(result), nil
		},
	)

	// Tool: create_article
	s.AddTool(
		mcp.NewTool("create_article",
			mcp.WithDescription("Create a new Knowledge Base article in a YouTrack project."),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project short name (key), e.g. CS, BACKEND"),
			),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("Article title"),
			),
			mcp.WithString("content",
				mcp.Description("Article content (markdown supported)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, _ := req.Params.Arguments["project_id"].(string)
			title, _ := req.Params.Arguments["title"].(string)
			content, _ := req.Params.Arguments["content"].(string)

			if projectID == "" || title == "" {
				return mcp.NewToolResultError("project_id and title are required"), nil
			}

			article, err := client.CreateArticle(ctx, projectID, title, content)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create article: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Created article %s: %s\n%s", article.ID, article.Title, article.URL)), nil
		},
	)

	if addr := os.Getenv("YOUTRACK_MCP_ADDR"); addr != "" {
		log.Printf("Starting YouTrack MCP server (SSE) on %s...", addr)
		sse := server.NewSSEServer(s, server.WithBaseURL("http://"+addr))
		if err := sse.Start(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Println("Starting YouTrack MCP server (stdio)...")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
