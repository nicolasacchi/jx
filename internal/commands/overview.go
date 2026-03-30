package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/spf13/cobra"
)

var overviewProjects string

func init() {
	rootCmd.AddCommand(overviewCmd)
	overviewCmd.Flags().StringVar(&overviewProjects, "project", "MLF", "Projects to scan (comma-separated)")
}

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Parallel project health snapshot",
	Long: `Fetch project health metrics in parallel: recent issues by status,
blocked items, and sprint progress.

Examples:
  jx overview
  jx overview --project MLF,SDD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		result := map[string]any{}
		var mu sync.Mutex
		var wg sync.WaitGroup

		project := overviewProjects

		// Issues updated in last 24h, grouped by status
		wg.Add(1)
		go func() {
			defer wg.Done()
			params := url.Values{
				"jql":        {fmt.Sprintf("project = %s AND updated >= -1d ORDER BY updated DESC", project)},
				"maxResults": {"100"},
				"fields":     {"status"},
			}
			data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
			if err != nil {
				mu.Lock()
				result["issues_24h_error"] = err.Error()
				mu.Unlock()
				return
			}
			var resp struct {
				Issues []struct {
					Fields struct {
						Status struct {
							Name string `json:"name"`
						} `json:"status"`
					} `json:"fields"`
				} `json:"issues"`
			}
			counts := map[string]int{}
			if json.Unmarshal(data, &resp) == nil {
				for _, issue := range resp.Issues {
					counts[issue.Fields.Status.Name]++
				}
			}
			mu.Lock()
			result["issues_updated_24h"] = counts
			result["issues_updated_24h_total"] = len(resp.Issues)
			mu.Unlock()
		}()

		// Blocked issues
		wg.Add(1)
		go func() {
			defer wg.Done()
			params := url.Values{
				"jql":        {fmt.Sprintf(`project = %s AND status = Blocked`, project)},
				"maxResults": {"50"},
				"fields":     {defaultSearchFields},
			}
			data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
			if err != nil {
				mu.Lock()
				result["blocked_error"] = err.Error()
				mu.Unlock()
				return
			}
			var resp struct {
				Issues []json.RawMessage `json:"issues"`
			}
			json.Unmarshal(data, &resp)
			mu.Lock()
			result["blocked_count"] = len(resp.Issues)
			mu.Unlock()
		}()

		// In Progress issues count
		wg.Add(1)
		go func() {
			defer wg.Done()
			params := url.Values{
				"jql":        {fmt.Sprintf(`project = %s AND status = "In Progress"`, project)},
				"maxResults": {"1"},
				"fields":     {"status"},
			}
			data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
			if err != nil {
				mu.Lock()
				result["in_progress_error"] = err.Error()
				mu.Unlock()
				return
			}
			var resp struct {
				Issues []json.RawMessage `json:"issues"`
				IsLast bool              `json:"isLast"`
			}
			json.Unmarshal(data, &resp)
			count := len(resp.Issues)
			if !resp.IsLast {
				count = -1 // more than maxResults
			}
			mu.Lock()
			result["in_progress_count"] = count
			mu.Unlock()
		}()

		wg.Wait()

		result["project"] = project
		out, _ := json.Marshal(result)
		return printData("", out)
	},
}
