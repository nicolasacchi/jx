# jx — Jira Explorer CLI

Purpose-built Go CLI for Jira Cloud, designed for Claude Code agent integration.

## Architecture

Follows the ddx/kv/gumlet pattern:

```
cmd/jx/main.go              → entry point
internal/client/             → Jira REST API v3 HTTP client (Basic Auth)
internal/commands/           → cobra commands (one file per resource group)
internal/config/             → TOML multi-project config (~/.config/jx/config.toml)
internal/output/             → auto-JSON on pipe, gjson filtering, TTY tables
internal/adf/                → Atlassian Document Format builder + markdown converter
internal/jql/                → fluent JQL builder from CLI flags
```

## Key Conventions

- **Auto-JSON**: JSON when piped (non-TTY), tables when interactive
- **gjson filtering**: `--jq` flag uses gjson syntax (NOT jq)
- **Exit codes**: 0=ok, 1=API error, 2=usage, 3=auth, 4=not found
- **Stderr**: total counts, deep links, warnings — never mixed with data
- **No interactive mode**: every command is non-interactive by default
- **ADF-native**: comments and descriptions accept markdown via `--file` flag, auto-converted to ADF

## Dual API

- Platform REST API v3: `{server}/rest/api/3/...` (issues, comments, fields, search)
- Software REST API: `{server}/rest/agile/1.0/...` (boards, sprints, epics)

## Auth Resolution

1. `--token`/`--email`/`--server` flags
2. `JIRA_API_TOKEN`/`JIRA_EMAIL`/`JIRA_SERVER` env vars
3. `~/.config/jx/config.toml` (multi-project)

## Building

```bash
make build    # → bin/jx
make install  # → ~/go/bin/jx
make test     # → go test ./...
```

## Adding a New Command Group

1. Create `internal/commands/{resource}.go`
2. Follow the pattern in `issues.go` or `comments.go`
3. Register parent command in `init()`: `rootCmd.AddCommand({resource}Cmd)`
4. Add table columns in `internal/output/table.go`
5. Test with `go build ./... && ./bin/jx {resource} list`
