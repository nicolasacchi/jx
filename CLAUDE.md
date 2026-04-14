# CLAUDE.md — jx

Go CLI for Jira Cloud. Single binary, JSON output, API token auth. 28 command groups, markdown→ADF converter, flag-based JQL builder, parallel health snapshots. Designed for AI agent integration.

**API**: Jira REST API v3 + Jira Software REST API. Base URL from config: `https://{instance}.atlassian.net`.

## Authentication

Resolution order (first non-empty wins):

1. `--email` / `--token` / `--server` flags
2. `JIRA_EMAIL` / `JIRA_API_TOKEN` / `JIRA_SERVER` env vars
3. `~/.config/jx/config.toml` — project from `--project` flag, then `default_project`

### Multi-project config

```toml
default_project = "production"

[projects.production]
email = "nicola@1000farmacie.it"
token = "ATATT3xFfGF0..."
server = "https://1000farmacie.atlassian.net"
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | — | JIRA_EMAIL override |
| `--token` | — | JIRA_API_TOKEN override |
| `--server` | — | JIRA_SERVER override |
| `--project` | — | Named project from config |
| `--json` | false | Force JSON output |
| `--jq` | — | gjson path filter |
| `--limit` | 50 | Max results |
| `--verbose` | false | Print request/response to stderr |
| `--quiet` | false | Suppress non-error output |

## Commands

### issues

```bash
jx issues list --project MLF --status "In Progress" --limit 10
jx issues list --project MLF --sprint current --type Story
jx issues list --project MLF --component TECH --resolution unresolved
jx issues get MLF-5146
jx issues get MLF-5146 --jq '{key:key,status:status,sprint:sprint,storyPoints:storyPoints}'
jx issues create --project MLF --type Story --summary "Title" --description-file desc.md
jx issues create --project MLF --type Sub-task --parent MLF-5147 --summary "Phase 1"
jx issues edit MLF-5146 --summary "New title" --description-file desc.md
jx issues delete MLF-5146
jx issues assign MLF-5146 --user ACCOUNT_ID
jx issues assign MLF-5146 --unassign
jx issues changelog MLF-5146 --limit 10
```

**API**: `GET /rest/api/3/search/jql` (list), `GET /rest/api/3/issue/{key}` (get), `POST /rest/api/3/issue` (create), `PUT /rest/api/3/issue/{key}` (edit), `DELETE /rest/api/3/issue/{key}` (delete), `GET /rest/api/3/issue/{key}/changelog` (changelog)

**issues list flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | — | Filter by project key |
| `--status` | — | Filter by status |
| `--type` | — | Filter by issue type |
| `--assignee` | — | Filter by assignee ('me', 'unassigned', or name) |
| `--updated` | — | Updated since (e.g., -7d) |
| `--labels` | — | Filter by labels (comma-separated) |
| `--priority` | — | Filter by priority |
| `--parent` | — | Filter by parent key |
| `--sprint` | — | Filter by sprint (current, closed, future, or name) |
| `--component` | — | Filter by component |
| `--version` | — | Filter by fix version |
| `--epic` | — | Filter by epic key |
| `--resolution` | — | Filter by resolution (or 'unresolved') |
| `--order-by` | `updated DESC` | ORDER BY clause |

**Flattened issue fields** (always returned in JSON):

key, id, self, summary, status, type, priority, assignee, reporter, parent, resolution, duedate, created, updated, labels, fixVersions, affectedVersions, components, links, subtasks, timeTracking, votes, watches, sprint, storyPoints, epicLink, flagged, description

### comments

```bash
jx comments list MLF-5146
jx comments add MLF-5146 --body "Simple text"
jx comments add MLF-5146 --file comment.md          # Markdown → ADF
jx comments add MLF-5146 --adf-file raw.json         # Raw ADF JSON
jx comments edit COMMENT_ID --issue MLF-5146 --body "Updated"
jx comments delete COMMENT_ID --issue MLF-5146
```

**API**: `GET/POST /rest/api/3/issue/{key}/comment`, `PUT/DELETE .../comment/{id}`

### search

```bash
jx search "project = MLF AND status = 'In Progress'"
jx search --project MLF --status "In Progress" --updated -7d
jx search --project MLF --sprint current --component TECH
jx search --text "login error" --project MLF
```

Supports all `issues list` flags plus `--created`, `--text`, `--due-before`, `--due-after`.

### transitions

```bash
jx transitions list MLF-5146
jx transitions move MLF-5146 "In Progress"
jx transitions move MLF-5146 Done --comment "Shipped in PR #6966"
```

### context

```bash
jx context add MLF-5146 --file context.md
```

Posts: italic intro paragraph + code block with the file content. ADF-native.

### overview

```bash
jx overview
jx overview --project MLF
```

Parallel fetch: issues updated in 24h (by status), blocked count, in-progress count.

### sprints

```bash
jx sprints list --board 40
jx sprints active --board 40
jx sprints get 920
jx sprints issues 920
jx sprints create --board 40 --name "Sprint 46" --start 2026-04-07T08:00:00Z --end 2026-04-21T08:00:00Z
jx sprints start SPRINT_ID
jx sprints close SPRINT_ID
```

**API**: `GET/POST /rest/agile/1.0/sprint`, `GET /rest/agile/1.0/board/{id}/sprint`

### boards

```bash
jx boards list --project MLF
jx boards get 40
jx boards config 40
```

### epics

```bash
jx epics list --project MLF
jx epics get MLF-100
jx epics issues MLF-100
```

### versions

```bash
jx versions list --project MLF
jx versions get VERSION_ID
jx versions create --project MLF --name "v2.1" --release-date 2026-04-15
jx versions release VERSION_ID
jx versions delete VERSION_ID
```

### components

```bash
jx components list --project MLF
jx components create --project MLF --name "Backend"
jx components delete COMPONENT_ID
```

### statuses

```bash
jx statuses list
jx statuses list --project MLF
```

### filters

```bash
jx filters list
jx filters get FILTER_ID
jx filters create --name "Blocked issues" --jql "project = MLF AND status = Blocked"
jx filters delete FILTER_ID
```

### permissions

```bash
jx permissions mine --project MLF
jx permissions check --project MLF --permission EDIT_ISSUES
```

### remote-links

```bash
jx remote-links list MLF-5146
jx remote-links create MLF-5146 --url "https://github.com/.../pull/6966" --title "PR #6966"
jx remote-links delete MLF-5146 LINK_ID
```

### backlog

```bash
jx backlog move MLF-5146,MLF-5147
jx backlog move-to-sprint 920 MLF-5146,MLF-5147
```

### votes, properties, worklogs, watchers, attachments, links

```bash
jx votes list MLF-5146
jx votes add MLF-5146
jx properties list MLF-5146
jx properties set MLF-5146 --key "deploy.status" --value '{"env":"prod"}'
jx worklogs add MLF-5146 --time 2h --comment "Code review"
jx watchers list MLF-5146
jx attachments list MLF-5146
jx attachments add MLF-5146 --file screenshot.png
jx attachments get 32175 --output ./downloads/                    # download single binary by ID
jx attachments download-all MLF-5146 --output ./attachments/      # download every attachment into per-issue subdir
jx links create MLF-5146 MLF-5145 --type "is blocked by"
```

`attachments get` / `download-all` skip files that already exist on disk (idempotent re-runs) and use a separate 5-minute HTTP timeout for binary content (the standard 30 s API timeout would cut off large videos/archives).

### service-desk

```bash
jx service-desk list --updated -2d
jx sd list --type Bug
jx sd create --summary "App crashes on login" --description-file bug.md
```

### bulk

```bash
jx bulk edit --jql "project = MLF AND sprint = 42" --set-labels "v2.1"
jx bulk move --jql "project = MLF AND status = 'Code Review'" --status Done
```

### audit, projects, users, fields, labels, open, config

```bash
jx audit list --from 2026-03-23
jx projects list
jx users me
jx fields list --custom --search "story"
jx labels list
jx open MLF-5146
jx config add production --email user@co.com --token TOKEN --server https://...
```

### export (migration bulk extract)

```bash
jx export --project MLF --output ./mlf-export/                                  # all issues
jx export --project MLF --updated -180d --output ./mlf-export/                  # recent only
jx export --project SDD --updated -90d --output ./sdd-export/ --include comments
jx export --project MLF --output ./mlf-export/ --download-attachments           # also fetch binaries
```

Produces:

- `<output>/issues.jsonl` — flattened issues, one per line, descriptions converted ADF→markdown
- `<output>/comments/<KEY>.json` — comments per open issue (skipped for Done/Resolved/Rejected/WON'T DO)
- `<output>/attachments.json` — attachment metadata for all issues (filename, size, content URL, author, created)
- `<output>/attachments/<KEY>/<filename>` — raw binaries (only with `--download-attachments`)

`--include` defaults to `comments,attachments`. Use `--include comments` to skip attachments metadata, `--include attachments` to skip comments. `--download-attachments` is idempotent — existing files are skipped on re-run, and downloads use the 5-minute binary client timeout.

## Markdown → ADF Converter

`internal/adf/markdown.go` converts a subset of markdown to Jira's Atlassian Document Format:

| Markdown | ADF Node |
|----------|----------|
| `# Heading` (1-6) | heading |
| Paragraphs | paragraph |
| ` ``` ` code blocks | codeBlock (with language) |
| `- item` / `* item` | bulletList |
| `1. item` | orderedList |
| `**bold**` | text + strong mark |
| `*italic*` | text + em mark |
| `` `code` `` | text + code mark |
| `[text](url)` | text + link mark |
| `> quote` | blockquote |
| `---` | rule |

## JQL Builder

`internal/jql/builder.go` constructs JQL from structured inputs:

```go
q := jql.New().
    Project("MLF").
    Status("In Progress").
    Sprint("current").
    Component("Backend").
    OrderBy("updated", "DESC").
    Build()
// → project = MLF AND status = "In Progress" AND sprint in openSprints() AND component = "Backend" ORDER BY updated DESC
```

## Output

- **TTY**: Tables (go-pretty) for commands with table definitions, JSON otherwise
- **Piped**: Always JSON
- `--json`: Force JSON on TTY
- `--jq`: gjson filter (NOT jq syntax). Array: `#.field`. Object: `#.{a:a,b:b}`

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API/network error |
| 3 | Auth error (401/403) |
| 4 | Not found (404) |

## Architecture

```
cmd/jx/main.go              Entry point, version injection, exit codes
internal/
  client/client.go           HTTP client, Basic Auth, retries, dual API
  client/errors.go           APIError with ExitCode(), hints
  commands/root.go           Root command, global flags, getClient()
  commands/*.go              One file per command group (33 files)
  config/config.go           TOML config, multi-project, credential resolution
  output/output.go           JSON/table dispatcher, TTY detection
  output/table.go            go-pretty table rendering, column definitions
  output/filter.go           gjson --jq filter
  adf/adf.go                 Fluent ADF document builder
  adf/markdown.go            Markdown → ADF converter
  adf/adf_test.go            Builder + converter tests
  jql/builder.go             Fluent JQL builder from flags
  jql/builder_test.go        JQL builder tests
```

## HTTP Client

- **Auth**: Basic Auth (base64 email:token)
- **Base URL**: `{server}/rest/api/3/` (Platform) or `{server}/rest/agile/1.0/` (Software)
- **Timeout**: 30s per request
- **Retries**: Max 3 on 429 (rate limit), exponential backoff (1s, 2s, 4s)
- **Error parsing**: Handles Jira's `{errorMessages: [...], errors: {...}}` format

## Adding a New Command Group

1. Create `internal/commands/{resource}.go`
2. Follow the pattern in `issues.go` or `comments.go`
3. Register parent command in `init()`: `rootCmd.AddCommand({resource}Cmd)`
4. Add table columns in `internal/output/table.go`
5. Test with `go build ./... && ./bin/jx {resource} list`
