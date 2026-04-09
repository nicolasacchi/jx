package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nicolasacchi/jx/internal/adf"
	"github.com/spf13/cobra"
)

var (
	exportProject string
	exportOutput  string
	exportUpdated string
	exportInclude string // "comments,attachments" or subset
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportProject, "project", "", "Project key (required)")
	exportCmd.MarkFlagRequired("project")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output directory (required)")
	exportCmd.MarkFlagRequired("output")
	exportCmd.Flags().StringVar(&exportUpdated, "updated", "", "Only issues updated since (e.g., -180d, -7d)")
	exportCmd.Flags().StringVar(&exportInclude, "include", "comments,attachments", "Data to include: comments,attachments (comma-separated)")
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export project issues for migration",
	Long: `Export all issues from a Jira project with descriptions converted to markdown.

Produces:
  <output>/issues.jsonl          — One flattened issue per line
  <output>/comments/<KEY>.json   — Comments per issue (open issues only)
  <output>/attachments.json      — Attachment metadata for all issues

Examples:
  jx export --project MLF --output ./mlf-export/
  jx export --project MLF --updated -180d --output ./mlf-export/
  jx export --project SDD --updated -90d --output ./sdd-export/ --include comments`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		includes := parseIncludes(exportInclude)

		// Create output directory
		if err := os.MkdirAll(exportOutput, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		// Phase 1: Export issues with pagination
		fmt.Fprintf(os.Stderr, "export: extracting %s issues...\n", exportProject)

		issuesFile, err := os.Create(filepath.Join(exportOutput, "issues.jsonl"))
		if err != nil {
			return fmt.Errorf("create issues.jsonl: %w", err)
		}
		defer issuesFile.Close()

		var allIssues []exportedIssue
		var nextPageToken string
		totalExported := 0
		pageSize := 100

		for {
			jqlStr := fmt.Sprintf("project = %s", exportProject)
			if exportUpdated != "" {
				jqlStr += fmt.Sprintf(" AND updated >= %q", exportUpdated)
			}
			jqlStr += " ORDER BY created ASC"

			params := url.Values{
				"jql":        {jqlStr},
				"maxResults": {strconv.Itoa(pageSize)},
				"fields":     {defaultSearchFields + ",attachment"},
			}
			if nextPageToken != "" {
				params.Set("nextPageToken", nextPageToken)
			}

			data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
			if err != nil {
				return fmt.Errorf("search page: %w", err)
			}

			var searchResp struct {
				Issues        []json.RawMessage `json:"issues"`
				Total         int               `json:"total"`
				NextPageToken string            `json:"nextPageToken"`
			}
			if err := json.Unmarshal(data, &searchResp); err != nil {
				return fmt.Errorf("parse response: %w", err)
			}

			if totalExported == 0 && searchResp.Total > 0 {
				fmt.Fprintf(os.Stderr, "export: %d total issues found\n", searchResp.Total)
			}

			for _, raw := range searchResp.Issues {
				flat := flattenIssue(raw)
				if flat == nil {
					continue
				}

				// Convert description to markdown
				if desc, ok := flat["description"].(json.RawMessage); ok {
					flat["description"] = adf.ToMarkdownFromJSON(desc)
				}

				// Extract attachment metadata before removing from flat output
				var attachments []attachmentMeta
				if includes["attachments"] {
					attachments = extractAttachments(raw)
				}

				key, _ := flat["key"].(string)
				status, _ := flat["status"].(string)

				ei := exportedIssue{
					key:         key,
					status:      status,
					attachments: attachments,
				}
				allIssues = append(allIssues, ei)

				// Write to JSONL
				line, _ := json.Marshal(flat)
				issuesFile.Write(line)
				issuesFile.WriteString("\n")
				totalExported++
			}

			if searchResp.Total > 0 {
				fmt.Fprintf(os.Stderr, "export: %d/%d issues exported\r", totalExported, searchResp.Total)
			} else {
				fmt.Fprintf(os.Stderr, "export: %d issues exported\r", totalExported)
			}

			if searchResp.NextPageToken == "" || len(searchResp.Issues) == 0 {
				break
			}
			nextPageToken = searchResp.NextPageToken
		}

		fmt.Fprintf(os.Stderr, "\nexport: %d issues written to issues.jsonl\n", totalExported)

		// Phase 2: Export comments for open issues
		if includes["comments"] {
			commentsDir := filepath.Join(exportOutput, "comments")
			if err := os.MkdirAll(commentsDir, 0755); err != nil {
				return fmt.Errorf("create comments dir: %w", err)
			}

			doneStatuses := map[string]bool{
				"Done": true, "WON'T DO": true, "Resolved": true, "Rejected": true,
			}

			openCount := 0
			commentCount := 0
			for _, ei := range allIssues {
				if doneStatuses[ei.status] {
					continue
				}
				openCount++

				data, err := c.Get(context.Background(), "rest/api/3/issue/"+ei.key+"/comment", nil)
				if err != nil {
					fmt.Fprintf(os.Stderr, "export: warning: comments for %s: %v\n", ei.key, err)
					continue
				}

				var commentResp struct {
					Comments []json.RawMessage `json:"comments"`
					Total    int               `json:"total"`
				}
				if json.Unmarshal(data, &commentResp) != nil || commentResp.Total == 0 {
					continue
				}

				// Convert comment bodies to markdown
				var comments []map[string]any
				for _, raw := range commentResp.Comments {
					var c struct {
						ID      string          `json:"id"`
						Body    json.RawMessage `json:"body"`
						Author  struct{ DisplayName string } `json:"author"`
						Created string `json:"created"`
						Updated string `json:"updated"`
					}
					if json.Unmarshal(raw, &c) != nil {
						continue
					}
					comments = append(comments, map[string]any{
						"id":      c.ID,
						"author":  c.Author.DisplayName,
						"body":    adf.ToMarkdownFromJSON(c.Body),
						"created": c.Created,
						"updated": c.Updated,
					})
				}

				out, _ := json.MarshalIndent(comments, "", "  ")
				outPath := filepath.Join(commentsDir, ei.key+".json")
				if err := os.WriteFile(outPath, out, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "export: warning: write %s: %v\n", outPath, err)
				}
				commentCount += len(comments)
			}
			fmt.Fprintf(os.Stderr, "export: %d comments from %d open issues\n", commentCount, openCount)
		}

		// Phase 3: Write attachment metadata
		if includes["attachments"] {
			var allAttachments []map[string]any
			for _, ei := range allIssues {
				for _, att := range ei.attachments {
					allAttachments = append(allAttachments, map[string]any{
						"issueKey": ei.key,
						"id":       att.id,
						"filename": att.filename,
						"size":     att.size,
						"mimeType": att.mimeType,
						"content":  att.contentURL,
						"author":   att.author,
						"created":  att.created,
					})
				}
			}

			out, _ := json.MarshalIndent(allAttachments, "", "  ")
			outPath := filepath.Join(exportOutput, "attachments.json")
			if err := os.WriteFile(outPath, out, 0644); err != nil {
				return fmt.Errorf("write attachments.json: %w", err)
			}
			fmt.Fprintf(os.Stderr, "export: %d attachments across %d issues\n",
				len(allAttachments), countIssuesWithAttachments(allIssues))
		}

		fmt.Fprintf(os.Stderr, "export: done. Output in %s\n", exportOutput)
		return nil
	},
}

type exportedIssue struct {
	key         string
	status      string
	attachments []attachmentMeta
}

type attachmentMeta struct {
	id         string
	filename   string
	size       int64
	mimeType   string
	contentURL string
	author     string
	created    string
}

func extractAttachments(raw json.RawMessage) []attachmentMeta {
	var issue struct {
		Fields struct {
			Attachment []struct {
				ID       string `json:"id"`
				Filename string `json:"filename"`
				Size     int64  `json:"size"`
				MimeType string `json:"mimeType"`
				Content  string `json:"content"`
				Created  string `json:"created"`
				Author   struct{ DisplayName string } `json:"author"`
			} `json:"attachment"`
		} `json:"fields"`
	}
	if json.Unmarshal(raw, &issue) != nil {
		return nil
	}
	var metas []attachmentMeta
	for _, a := range issue.Fields.Attachment {
		metas = append(metas, attachmentMeta{
			id:         a.ID,
			filename:   a.Filename,
			size:       a.Size,
			mimeType:   a.MimeType,
			contentURL: a.Content,
			author:     a.Author.DisplayName,
			created:    a.Created,
		})
	}
	return metas
}

func countIssuesWithAttachments(issues []exportedIssue) int {
	count := 0
	for _, ei := range issues {
		if len(ei.attachments) > 0 {
			count++
		}
	}
	return count
}

func parseIncludes(s string) map[string]bool {
	m := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		m[strings.TrimSpace(part)] = true
	}
	return m
}
