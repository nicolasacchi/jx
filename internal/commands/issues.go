package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/nicolasacchi/jx/internal/adf"
	"github.com/nicolasacchi/jx/internal/jql"
	"github.com/spf13/cobra"
)

var (
	issuesProject    string
	issuesStatus     string
	issuesType       string
	issuesAssignee   string
	issuesUpdated    string
	issuesLabels     string
	issuesPriority   string
	issuesParent     string
	issuesSprint     string
	issuesOrderBy    string
	issuesFields     string
	issuesSummary    string
	issuesDescFile   string
	issuesAssignUser string
	issuesComponent  string
	issuesVersion    string
	issuesEpic       string
	issuesResolution string
	createType       string // separate from issuesType to avoid default leaking to list
)

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesGetCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesEditCmd)
	issuesCmd.AddCommand(issuesDeleteCmd)
	issuesCmd.AddCommand(issuesAssignCmd)
	issuesCmd.AddCommand(issuesChangelogCmd)

	// list flags
	issuesListCmd.Flags().StringVar(&issuesProject, "project", "", "Filter by project key (e.g., MLF)")
	issuesListCmd.Flags().StringVar(&issuesStatus, "status", "", "Filter by status")
	issuesListCmd.Flags().StringVar(&issuesType, "type", "", "Filter by issue type (Story, Bug, Sub-task, etc.)")
	issuesListCmd.Flags().StringVar(&issuesAssignee, "assignee", "", "Filter by assignee (name, 'me', 'unassigned')")
	issuesListCmd.Flags().StringVar(&issuesUpdated, "updated", "", "Updated since (e.g., -7d, -1h)")
	issuesListCmd.Flags().StringVar(&issuesLabels, "labels", "", "Filter by labels (comma-separated)")
	issuesListCmd.Flags().StringVar(&issuesPriority, "priority", "", "Filter by priority")
	issuesListCmd.Flags().StringVar(&issuesParent, "parent", "", "Filter by parent key (for subtasks)")
	issuesListCmd.Flags().StringVar(&issuesSprint, "sprint", "", "Filter by sprint (current, closed, future, or name)")
	issuesListCmd.Flags().StringVar(&issuesComponent, "component", "", "Filter by component")
	issuesListCmd.Flags().StringVar(&issuesVersion, "version", "", "Filter by fix version")
	issuesListCmd.Flags().StringVar(&issuesEpic, "epic", "", "Filter by epic key")
	issuesListCmd.Flags().StringVar(&issuesResolution, "resolution", "", "Filter by resolution (or 'unresolved')")
	issuesListCmd.Flags().StringVar(&issuesOrderBy, "order-by", "updated DESC", "ORDER BY clause")

	// get flags
	issuesGetCmd.Flags().StringVar(&issuesFields, "fields", "", "Comma-separated fields to return")

	// create flags
	issuesCreateCmd.Flags().StringVar(&issuesProject, "project", "", "Project key (required)")
	issuesCreateCmd.Flags().StringVar(&createType, "type", "Story", "Issue type")
	issuesCreateCmd.Flags().StringVar(&issuesSummary, "summary", "", "Issue summary (required)")
	issuesCreateCmd.Flags().StringVar(&issuesDescFile, "description-file", "", "Markdown file for description (converted to ADF)")
	issuesCreateCmd.Flags().StringVar(&issuesParent, "parent", "", "Parent issue key (for subtasks)")
	issuesCreateCmd.Flags().StringVar(&issuesLabels, "labels", "", "Labels (comma-separated)")
	issuesCreateCmd.Flags().StringVar(&issuesPriority, "priority", "", "Priority (e.g., High, Medium, Low)")
	issuesCreateCmd.Flags().StringVar(&issuesAssignee, "assignee", "", "Assignee account ID or display name")
	issuesCreateCmd.MarkFlagRequired("project")
	issuesCreateCmd.MarkFlagRequired("summary")

	// edit flags
	issuesEditCmd.Flags().StringVar(&issuesSummary, "summary", "", "New summary")
	issuesEditCmd.Flags().StringVar(&issuesDescFile, "description-file", "", "Markdown file for description (converted to ADF)")
	issuesEditCmd.Flags().StringVar(&issuesLabels, "labels", "", "Set labels (comma-separated)")
	issuesEditCmd.Flags().StringVar(&issuesPriority, "priority", "", "Set priority")

	// assign flags
	issuesAssignCmd.Flags().StringVar(&issuesAssignUser, "user", "", "Assignee account ID")
	issuesAssignCmd.Flags().BoolP("unassign", "u", false, "Remove assignee")
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage Jira issues",
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues with JQL filtering",
	Long: `List issues matching filter criteria. Flags are combined into JQL.

Examples:
  jx issues list --project MLF --limit 10
  jx issues list --project MLF --status "In Progress" --assignee me
  jx issues list --project MLF --type Bug --updated -7d
  jx issues list --project MLF --sprint current`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		var labels []string
		if issuesLabels != "" {
			labels = strings.Split(issuesLabels, ",")
		}

		orderField, orderDir := parseOrderBy(issuesOrderBy)

		q := jql.New().
			Project(issuesProject).
			Status(issuesStatus).
			Type(issuesType).
			Assignee(issuesAssignee).
			UpdatedSince(issuesUpdated).
			Labels(labels).
			Priority(issuesPriority).
			Parent(issuesParent).
			Sprint(issuesSprint).
			Component(issuesComponent).
			FixVersion(issuesVersion).
			Epic(issuesEpic).
			Resolution(issuesResolution).
			OrderBy(orderField, orderDir)

		jqlStr := q.Build()
		if jqlStr == "" {
			jqlStr = "ORDER BY updated DESC"
		}

		params := url.Values{
			"jql":        {jqlStr},
			"maxResults": {strconv.Itoa(limitFlag)},
			"fields":     {defaultSearchFields},
		}

		data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
		if err != nil {
			return err
		}

		return printData("issues.list", flattenIssues(data))
	},
}

var issuesGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a single issue by key",
	Long: `Fetch full details for a Jira issue.

Examples:
  jx issues get MLF-5146
  jx issues get MLF-5146 --fields summary,status,assignee`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{}
		if issuesFields != "" {
			params.Set("fields", issuesFields)
		}

		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0], params)
		if err != nil {
			return err
		}

		if flat := flattenIssue(data); flat != nil {
			out, _ := json.Marshal(flat)
			return printData("issues.get", out)
		}
		return printData("", data)
	},
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Long: `Create a Jira issue. Description can be provided as a markdown file.

Examples:
  jx issues create --project MLF --type Story --summary "Fix login bug"
  jx issues create --project MLF --type Sub-task --parent MLF-5147 --summary "Phase 1"
  jx issues create --project MLF --type Story --summary "Cache fix" --description-file desc.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		fields := map[string]any{
			"project":   map[string]any{"key": issuesProject},
			"summary":   issuesSummary,
			"issuetype": map[string]any{"name": createType},
		}

		if issuesParent != "" {
			fields["parent"] = map[string]any{"key": issuesParent}
		}
		if issuesPriority != "" {
			fields["priority"] = map[string]any{"name": issuesPriority}
		}
		if issuesLabels != "" {
			fields["labels"] = strings.Split(issuesLabels, ",")
		}

		if issuesDescFile != "" {
			desc, err := readDescriptionFile(issuesDescFile)
			if err != nil {
				return err
			}
			fields["description"] = desc
		}

		body := map[string]any{"fields": fields}
		data, err := c.Post(context.Background(), "rest/api/3/issue", body)
		if err != nil {
			return err
		}

		// Print the created issue key
		var created struct {
			Key  string `json:"key"`
			ID   string `json:"id"`
			Self string `json:"self"`
		}
		if json.Unmarshal(data, &created) == nil && created.Key != "" {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "created: %s (%s)\n", created.Key, buildBrowseURL(c.Server(), created.Key))
			}
		}

		return printData("", data)
	},
}

var issuesEditCmd = &cobra.Command{
	Use:   "edit <key>",
	Short: "Edit an issue's fields",
	Long: `Edit fields on an existing issue.

Examples:
  jx issues edit MLF-5146 --summary "Updated title"
  jx issues edit MLF-5146 --description-file desc.md
  jx issues edit MLF-5146 --priority High --labels "v2.1,urgent"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		fields := map[string]any{}

		if issuesSummary != "" {
			fields["summary"] = issuesSummary
		}
		if issuesPriority != "" {
			fields["priority"] = map[string]any{"name": issuesPriority}
		}
		if issuesLabels != "" {
			fields["labels"] = strings.Split(issuesLabels, ",")
		}
		if issuesDescFile != "" {
			desc, err := readDescriptionFile(issuesDescFile)
			if err != nil {
				return err
			}
			fields["description"] = desc
		}

		if len(fields) == 0 {
			return fmt.Errorf("no fields to update; use --summary, --description-file, --priority, or --labels")
		}

		body := map[string]any{"fields": fields}
		_, err = c.Put(context.Background(), "rest/api/3/issue/"+args[0], body)
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "updated: %s\n", args[0])
		}
		return nil
	},
}

var issuesDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/issue/"+args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "deleted: %s\n", args[0])
		}
		return nil
	},
}

var issuesAssignCmd = &cobra.Command{
	Use:   "assign <key>",
	Short: "Assign or unassign an issue",
	Long: `Assign an issue to a user or remove assignment.

Examples:
  jx issues assign MLF-5146 --user 557058:12345678-abcd-...
  jx issues assign MLF-5146 --unassign`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		unassign, _ := cmd.Flags().GetBool("unassign")

		var body map[string]any
		if unassign {
			body = map[string]any{"accountId": nil}
		} else if issuesAssignUser != "" {
			body = map[string]any{"accountId": issuesAssignUser}
		} else {
			return fmt.Errorf("specify --user or --unassign")
		}

		_, err = c.Put(context.Background(), "rest/api/3/issue/"+args[0]+"/assignee", body)
		if err != nil {
			return err
		}

		if !quietFlag {
			if unassign {
				fmt.Fprintf(os.Stderr, "unassigned: %s\n", args[0])
			} else {
				fmt.Fprintf(os.Stderr, "assigned: %s → %s\n", args[0], issuesAssignUser)
			}
		}
		return nil
	},
}

// readDescriptionFile reads a markdown file and converts to an ADF document.
func readDescriptionFile(path string) (*adf.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read description file: %w", err)
	}
	return adf.FromMarkdown(string(content)), nil
}

// parseOrderBy splits "field direction" into field and direction.
func parseOrderBy(s string) (string, string) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], "DESC"
	}
	return parts[0], parts[1]
}

var issuesChangelogCmd = &cobra.Command{
	Use:   "changelog <key>",
	Short: "Show issue change history",
	Long: `Display the changelog for an issue — who changed what and when.

Examples:
  jx issues changelog MLF-5146
  jx issues changelog MLF-5146 --limit 10`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{"maxResults": {strconv.Itoa(limitFlag)}}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/changelog", params)
		if err != nil {
			return err
		}

		var resp struct {
			Values []json.RawMessage `json:"values"`
			Total  int               `json:"total"`
		}
		if json.Unmarshal(data, &resp) == nil && resp.Values != nil {
			if resp.Total > 0 {
				fmt.Fprintf(os.Stderr, "changelog: %d total entries\n", resp.Total)
			}
			var flattened []map[string]any
			for _, raw := range resp.Values {
				var entry struct {
					ID      string `json:"id"`
					Author  struct{ DisplayName string } `json:"author"`
					Created string `json:"created"`
					Items   []struct {
						Field      string `json:"field"`
						FromString string `json:"fromString"`
						ToString   string `json:"toString"`
					} `json:"items"`
				}
				if json.Unmarshal(raw, &entry) != nil {
					continue
				}
				for _, item := range entry.Items {
					flattened = append(flattened, map[string]any{
						"id":      entry.ID,
						"author":  entry.Author.DisplayName,
						"created": entry.Created,
						"field":   item.Field,
						"from":    item.FromString,
						"to":      item.ToString,
					})
				}
			}
			if flattened == nil {
				flattened = []map[string]any{}
			}
			out, _ := json.Marshal(flattened)
			return printData("", out)
		}
		return printData("", data)
	},
}
