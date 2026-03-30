package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusesProject string

func init() {
	rootCmd.AddCommand(statusesCmd)
	statusesCmd.AddCommand(statusesListCmd)

	statusesListCmd.Flags().StringVar(&statusesProject, "project", "", "Filter statuses by project (shows per-issuetype)")
}

var statusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "List workflow statuses",
}

var statusesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all statuses",
	Long: `List Jira workflow statuses. Without --project, shows all statuses.
With --project, shows statuses grouped by issue type for that project.

Examples:
  jx statuses list
  jx statuses list --project MLF`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		if statusesProject != "" {
			data, err := c.Get(context.Background(), "rest/api/3/project/"+statusesProject+"/statuses", nil)
			if err != nil {
				return err
			}
			// Returns [{name: "Story", statuses: [{id, name, statusCategory}]}, ...]
			var issueTypes []struct {
				Name     string `json:"name"`
				Statuses []struct {
					ID             string `json:"id"`
					Name           string `json:"name"`
					StatusCategory struct{ Name string } `json:"statusCategory"`
				} `json:"statuses"`
			}
			if json.Unmarshal(data, &issueTypes) == nil {
				var rows []map[string]any
				for _, it := range issueTypes {
					for _, s := range it.Statuses {
						rows = append(rows, map[string]any{
							"id":        s.ID,
							"name":      s.Name,
							"category":  s.StatusCategory.Name,
							"issueType": it.Name,
						})
					}
				}
				if rows == nil {
					rows = []map[string]any{}
				}
				out, _ := json.Marshal(rows)
				return printData("statuses.list", out)
			}
			return printData("", data)
		}

		// All statuses
		data, err := c.Get(context.Background(), "rest/api/3/status", nil)
		if err != nil {
			return err
		}

		var statuses []json.RawMessage
		if json.Unmarshal(data, &statuses) == nil {
			fmt.Fprintf(os.Stderr, "statuses: %d\n", len(statuses))
			var rows []map[string]any
			for _, raw := range statuses {
				var s struct {
					ID             string `json:"id"`
					Name           string `json:"name"`
					StatusCategory struct{ Name string } `json:"statusCategory"`
				}
				if json.Unmarshal(raw, &s) != nil {
					continue
				}
				rows = append(rows, map[string]any{
					"id":       s.ID,
					"name":     s.Name,
					"category": s.StatusCategory.Name,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("statuses.list", out)
		}
		return printData("", data)
	},
}
