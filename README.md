# youtrack-mcp

MCP server for YouTrack REST API. Provides tools for reading and managing issues, comments, merge requests, and Knowledge Base articles.

## Tools

| Tool | Description |
|------|-------------|
| `get_issue_with_docs` | **Main.** Fetch an issue and all linked KB articles from its description |
| `get_issue` | Get an issue by ID |
| `get_issue_comments` | Get all comments on an issue |
| `get_issue_mrs` | Get merge requests linked to an issue (requires GitLab plugin) |
| `add_comment` | Add a comment to an issue |
| `update_issue` | Update an issue via YouTrack command syntax |
| `create_issue` | Create a new issue in a project |
| `search_issues` | Search issues using YouTrack query language |
| `list_project_issues` | List issues in a project |
| `get_article` | Get a KB article by ID or URL |
| `list_articles` | List KB articles in a project |
| `search_articles` | Search KB articles by text |
| `create_article` | Create a KB article |

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `YOUTRACK_URL` | Yes | Base URL of your YouTrack instance, e.g. `https://youtrack.example.com` |
| `YOUTRACK_TOKEN` | Yes | Personal permanent token |
| `YOUTRACK_GITLAB_PLUGIN` | No | Extension endpoint name for the GitLab integration plugin. Required for `get_issue_mrs`. Find it in your YouTrack URL: `/api/extensionEndpoints/<name>/backend/pullRequests` |
| `YOUTRACK_MCP_ADDR` | No | If set, starts in SSE/HTTP mode instead of stdio, e.g. `localhost:8080` |

## Auth token

YouTrack → your profile → **Hub** → **Auth tokens** → Create personal token.

Minimum scopes: `Read Issue`, `Read Article`. The `get_issue_mrs` tool uses a GitLab plugin extension endpoint — no special scope needed, but the plugin must be installed in your YouTrack instance.

## Connection options

### 1. Binary — stdio (Claude Desktop / Claude Code)

Build the binary:

```bash
go build -o youtrack-mcp .
```

**Claude Desktop** — `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "youtrack": {
      "command": "/path/to/youtrack-mcp",
      "env": {
        "YOUTRACK_URL": "https://youtrack.example.com",
        "YOUTRACK_TOKEN": "perm:xxxxxxxx"
      }
    }
  }
}
```

**Claude Code (global)** — `~/.claude/.claude.json` (the exact path depends on your Claude Code installation; check where your other MCP servers are configured):

```json
{
  "mcpServers": {
    "youtrack": {
      "command": "/path/to/youtrack-mcp",
      "env": {
        "YOUTRACK_URL": "https://youtrack.example.com",
        "YOUTRACK_TOKEN": "perm:xxxxxxxx"
      }
    }
  }
}
```

**Claude Code (project-level)** — `.claude/settings.json` in the project root (safe to commit, keeps token out via env):

```json
{
  "mcpServers": {
    "youtrack": {
      "command": "/path/to/youtrack-mcp",
      "env": {
        "YOUTRACK_URL": "https://youtrack.example.com",
        "YOUTRACK_TOKEN": "perm:xxxxxxxx"
      }
    }
  }
}
```

---

### 2. Docker — stdio

```bash
docker build -t youtrack-mcp .
```

**Claude Desktop / Claude Code:**

```json
{
  "mcpServers": {
    "youtrack": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "YOUTRACK_URL=https://youtrack.example.com",
        "-e", "YOUTRACK_TOKEN=perm:xxxxxxxx",
        "youtrack-mcp"
      ]
    }
  }
}
```

---

### 3. HTTP (Streamable HTTP — TCP)

Set `YOUTRACK_MCP_ADDR` to start an HTTP server instead of stdio. This is the recommended remote transport (SSE is deprecated).

**Run the server:**

```bash
YOUTRACK_URL=https://youtrack.example.com \
YOUTRACK_TOKEN=perm:xxxxxxxx \
YOUTRACK_MCP_ADDR=localhost:8080 \
./youtrack-mcp
```

Or with Docker:

```bash
docker run -p 8080:8080 \
  -e YOUTRACK_URL=https://youtrack.example.com \
  -e YOUTRACK_TOKEN=perm:xxxxxxxx \
  -e YOUTRACK_MCP_ADDR=0.0.0.0:8080 \
  youtrack-mcp
```

**Claude Desktop / Claude Code:**

```json
{
  "mcpServers": {
    "youtrack": {
      "type": "http",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

---

## How article auto-detection works

`get_issue_with_docs` scans the issue description for KB article references:

- Full URLs: `https://youtrack.example.com/articles/PROJECT-A-123`
- Bare IDs: `PROJECT-A-123` (pattern `[A-Z]+-A-\d+`)

All matched articles are fetched and appended to the response automatically.
