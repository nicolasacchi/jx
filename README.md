# jx

CLI for [Jira Cloud](https://www.atlassian.com/software/jira). 28 command groups covering issues, comments, search, transitions, sprints, boards, epics, versions, components, filters, permissions, bulk operations, service desk, and more — with markdown-to-ADF conversion, flag-based JQL builder, gjson filtering, and parallel health snapshots. Designed for AI agent integration.

## Install

```bash
go install github.com/nicolasacchi/jx/cmd/jx@latest
```

Or build from source:

```bash
git clone https://github.com/nicolasacchi/jx.git
cd jx
make install
```

## Quick Start

```bash
# Configure credentials
export JIRA_API_TOKEN=your-token
export JIRA_EMAIL=your-email@company.com
export JIRA_SERVER=https://your-instance.atlassian.net

# Or use config file
jx config add production --email user@company.com --token TOKEN --server https://your-instance.atlassian.net

# List issues
jx issues list --project MLF --limit 10

# Get a single issue (enriched: sprint, story points, links, subtasks, etc.)
jx issues get MLF-5146

# Search with flag-based JQL
jx search --project MLF --status "In Progress" --sprint current

# Search with raw JQL
jx search "project = MLF AND updated >= -7d ORDER BY updated DESC"

# Add a comment with markdown → ADF (code blocks work!)
jx comments add MLF-5146 --file comment.md

# Cross-instance context handoff
jx context add MLF-5146 --file context.md

# Transition an issue
jx transitions move MLF-5146 "In Progress"

# Project health overview (parallel fetch)
jx overview --project MLF

# Check your permissions
jx permissions mine --project MLF
```

## Features

### Auto-JSON Output

```bash
# TTY → human-readable table
jx issues list --project MLF

# Piped → auto-JSON (no flag needed)
jx issues list --project MLF | cat

# Force JSON + gjson filtering
jx issues get MLF-5146 --jq '{key:key,status:status,sprint:sprint}'
```

### Markdown → ADF Conversion

The `--file` flag on `comments add`, `context add`, and `issues create --description-file` converts markdown to Jira's Atlassian Document Format. Supports headings, paragraphs, fenced code blocks, bullet/ordered lists, bold, italic, inline code, links, blockquotes, and horizontal rules.

```bash
jx comments add MLF-5146 --file context.md
```

This was the original motivation for building jx — the previous CLI silently corrupted code blocks.

### Flag-Based JQL Builder

```bash
# These produce equivalent JQL:
jx search "project = MLF AND status = 'In Progress' AND updated >= '-7d'"
jx search --project MLF --status "In Progress" --updated -7d
```

Available flags: `--project`, `--status`, `--type`, `--assignee`, `--updated`, `--created`, `--labels`, `--priority`, `--sprint`, `--parent`, `--text`, `--component`, `--version`, `--epic`, `--resolution`, `--due-before`, `--due-after`.

### Cross-Instance Context Sharing

Post structured comments designed for another AI agent to consume:

```bash
jx context add MLF-5146 --file prompt.md
```

Creates a comment with an italic intro and a code block containing the full prompt text — properly formatted as ADF.

## Command Groups

| Group | Subcommands |
|-------|-------------|
| `issues` | list, get, create, edit, delete, assign, changelog |
| `comments` | list, add, edit, delete |
| `search` | (raw JQL or flag-based) |
| `transitions` | list, move |
| `context` | add |
| `overview` | (parallel health snapshot) |
| `sprints` | list, active, get, issues, create, start, close |
| `boards` | list, get, config |
| `epics` | list, get, issues |
| `versions` | list, get, create, release, delete |
| `components` | list, create, delete |
| `projects` | list, get |
| `users` | list, me |
| `statuses` | list |
| `filters` | list, get, create, delete |
| `fields` | list |
| `labels` | list |
| `links` | create, delete |
| `remote-links` | list, create, delete |
| `permissions` | mine, check |
| `backlog` | move, move-to-sprint |
| `votes` | list, add, remove |
| `properties` | list, get, set, delete |
| `worklogs` | list, add |
| `watchers` | list, add, remove |
| `attachments` | list, add, get, delete |
| `service-desk` | list, get, create |
| `bulk` | edit, move |
| `audit` | list |
| `open` | (browser / --url) |
| `config` | add, remove, list, use, current |

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | `JIRA_EMAIL` | Jira account email |
| `--token` | `JIRA_API_TOKEN` | Jira API token |
| `--server` | `JIRA_SERVER` | Jira server URL |
| `--project` | — | Named project from config |
| `--json` | false | Force JSON output |
| `--jq` | — | gjson filter path |
| `--limit` | 50 | Max results |
| `-v` | false | Verbose (request/response to stderr) |
| `-q` | false | Quiet (suppress non-error output) |

## Authentication

Three-tier resolution (first non-empty wins):

1. `--email`/`--token`/`--server` flags
2. `JIRA_EMAIL`/`JIRA_API_TOKEN`/`JIRA_SERVER` environment variables
3. `~/.config/jx/config.toml` multi-project config

```bash
jx config add production --email user@company.com --token TOKEN --server https://instance.atlassian.net
jx config use production
```

## Architecture

```
cmd/jx/main.go              Entry point, version injection
internal/
  client/client.go           HTTP client, Basic Auth, retries, error parsing
  client/errors.go           APIError with exit codes and hints
  commands/root.go           Root command, global flags, getClient()
  commands/*.go              One file per command group (33 files)
  config/config.go           TOML config, multi-project, credential resolution
  output/output.go           JSON/table dispatcher, TTY detection
  output/table.go            go-pretty table rendering, column definitions
  output/filter.go           gjson --jq filter
  adf/adf.go                 Fluent ADF document builder
  adf/markdown.go            Markdown → ADF converter
  jql/builder.go             Fluent JQL builder from CLI flags
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API error (400, 5xx) |
| 3 | Auth error (401, 403) |
| 4 | Not found (404) |

## License

MIT
