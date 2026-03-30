package commands

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/nicolasacchi/jx/internal/jql"
	"github.com/spf13/cobra"
)

var (
	searchProject    string
	searchStatus     string
	searchType       string
	searchAssignee   string
	searchUpdated    string
	searchCreated    string
	searchLabels     string
	searchPriority   string
	searchSprint     string
	searchParent     string
	searchText       string
	searchOrderBy    string
	searchComponent  string
	searchVersion    string
	searchEpic       string
	searchResolution string
	searchDueBefore  string
	searchDueAfter   string
)

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchProject, "project", "", "Filter by project key")
	searchCmd.Flags().StringVar(&searchStatus, "status", "", "Filter by status")
	searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by issue type")
	searchCmd.Flags().StringVar(&searchAssignee, "assignee", "", "Filter by assignee")
	searchCmd.Flags().StringVar(&searchUpdated, "updated", "", "Updated since (e.g., -7d)")
	searchCmd.Flags().StringVar(&searchCreated, "created", "", "Created since (e.g., -30d)")
	searchCmd.Flags().StringVar(&searchLabels, "labels", "", "Filter by labels (comma-separated)")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority")
	searchCmd.Flags().StringVar(&searchSprint, "sprint", "", "Filter by sprint (current, closed, future, or name)")
	searchCmd.Flags().StringVar(&searchParent, "parent", "", "Filter by parent key")
	searchCmd.Flags().StringVar(&searchText, "text", "", "Full-text search")
	searchCmd.Flags().StringVar(&searchComponent, "component", "", "Filter by component")
	searchCmd.Flags().StringVar(&searchVersion, "version", "", "Filter by fix version")
	searchCmd.Flags().StringVar(&searchEpic, "epic", "", "Filter by epic key")
	searchCmd.Flags().StringVar(&searchResolution, "resolution", "", "Filter by resolution (or 'unresolved')")
	searchCmd.Flags().StringVar(&searchDueBefore, "due-before", "", "Due date <= (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchDueAfter, "due-after", "", "Due date >= (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchOrderBy, "order-by", "updated DESC", "ORDER BY clause")
}

var searchCmd = &cobra.Command{
	Use:   "search [jql]",
	Short: "Search issues with JQL",
	Long: `Search Jira issues using raw JQL or structured flags.

If a positional argument is provided, it's used as raw JQL.
Otherwise, flags are combined into JQL automatically.

Examples:
  jx search "project = MLF AND status = 'In Progress'"
  jx search --project MLF --status "In Progress" --updated -7d
  jx search --project MLF --sprint current --type Story
  jx search --project SDD --type "Service Request" --updated -2d
  jx search --text "login error" --project MLF`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		var jqlStr string
		if len(args) > 0 {
			// Raw JQL mode
			jqlStr = strings.Join(args, " ")
		} else {
			// Build from flags
			var labels []string
			if searchLabels != "" {
				labels = strings.Split(searchLabels, ",")
			}

			orderField, orderDir := parseOrderBy(searchOrderBy)

			q := jql.New().
				Project(searchProject).
				Status(searchStatus).
				Type(searchType).
				Assignee(searchAssignee).
				UpdatedSince(searchUpdated).
				CreatedSince(searchCreated).
				Labels(labels).
				Priority(searchPriority).
				Sprint(searchSprint).
				Parent(searchParent).
				Text(searchText).
				Component(searchComponent).
				FixVersion(searchVersion).
				Epic(searchEpic).
				Resolution(searchResolution).
				DueBefore(searchDueBefore).
				DueAfter(searchDueAfter).
				OrderBy(orderField, orderDir)

			jqlStr = q.Build()
			if jqlStr == "" {
				jqlStr = "ORDER BY updated DESC"
			}
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

		return printData("search", flattenIssues(data))
	},
}
