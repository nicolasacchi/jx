package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	bulkJQL       string
	bulkSetLabels string
	bulkStatus    string
)

func init() {
	rootCmd.AddCommand(bulkCmd)
	bulkCmd.AddCommand(bulkEditCmd)
	bulkCmd.AddCommand(bulkMoveCmd)

	bulkEditCmd.Flags().StringVar(&bulkJQL, "jql", "", "JQL to select issues (required)")
	bulkEditCmd.Flags().StringVar(&bulkSetLabels, "set-labels", "", "Set labels (comma-separated)")
	bulkEditCmd.MarkFlagRequired("jql")

	bulkMoveCmd.Flags().StringVar(&bulkJQL, "jql", "", "JQL to select issues (required)")
	bulkMoveCmd.Flags().StringVar(&bulkStatus, "status", "", "Target status (required)")
	bulkMoveCmd.MarkFlagRequired("jql")
	bulkMoveCmd.MarkFlagRequired("status")
}

var bulkCmd = &cobra.Command{
	Use:   "bulk",
	Short: "Batch operations on multiple issues",
}

var bulkEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Bulk edit issues matching JQL",
	Long: `Edit fields on all issues matching a JQL query.

Examples:
  jx bulk edit --jql "project = MLF AND sprint = 42" --set-labels "v2.1"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		// First, find all matching issues
		params := url.Values{
			"jql":        {bulkJQL},
			"maxResults": {"100"},
			"fields":     {"key"},
		}
		data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
		if err != nil {
			return err
		}

		var resp struct {
			Issues []struct {
				Key string `json:"key"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}

		if len(resp.Issues) == 0 {
			fmt.Fprintln(os.Stderr, "no issues match the query")
			return nil
		}

		fmt.Fprintf(os.Stderr, "bulk edit: %d issues\n", len(resp.Issues))

		fields := map[string]any{}
		if bulkSetLabels != "" {
			fields["labels"] = strings.Split(bulkSetLabels, ",")
		}

		if len(fields) == 0 {
			return fmt.Errorf("no fields to update; use --set-labels")
		}

		body := map[string]any{"fields": fields}

		var success, fail int
		for _, issue := range resp.Issues {
			_, err := c.Put(context.Background(), "rest/api/3/issue/"+issue.Key, body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  failed: %s — %s\n", issue.Key, err)
				fail++
			} else {
				success++
			}
		}

		fmt.Fprintf(os.Stderr, "done: %d updated, %d failed\n", success, fail)
		return nil
	},
}

var bulkMoveCmd = &cobra.Command{
	Use:   "move",
	Short: "Bulk transition issues matching JQL",
	Long: `Transition all issues matching a JQL query to a new status.

Examples:
  jx bulk move --jql "project = MLF AND status = 'Code Review'" --status Done`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{
			"jql":        {bulkJQL},
			"maxResults": {"100"},
			"fields":     {"key"},
		}
		data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
		if err != nil {
			return err
		}

		var resp struct {
			Issues []struct {
				Key string `json:"key"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}

		if len(resp.Issues) == 0 {
			fmt.Fprintln(os.Stderr, "no issues match the query")
			return nil
		}

		fmt.Fprintf(os.Stderr, "bulk move: %d issues → %s\n", len(resp.Issues), bulkStatus)

		var success, fail int
		for _, issue := range resp.Issues {
			// Get available transitions for each issue
			tData, err := c.Get(context.Background(), "rest/api/3/issue/"+issue.Key+"/transitions", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  failed: %s — %s\n", issue.Key, err)
				fail++
				continue
			}

			var tResp struct {
				Transitions []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					To   struct {
						Name string `json:"name"`
					} `json:"to"`
				} `json:"transitions"`
			}
			if err := json.Unmarshal(tData, &tResp); err != nil {
				fail++
				continue
			}

			var transitionID string
			for _, t := range tResp.Transitions {
				if strings.EqualFold(t.Name, bulkStatus) || strings.EqualFold(t.To.Name, bulkStatus) {
					transitionID = t.ID
					break
				}
			}
			if transitionID == "" {
				fmt.Fprintf(os.Stderr, "  skipped: %s — no transition to %q\n", issue.Key, bulkStatus)
				fail++
				continue
			}

			body := map[string]any{
				"transition": map[string]any{"id": transitionID},
			}
			_, err = c.Post(context.Background(), "rest/api/3/issue/"+issue.Key+"/transitions", body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  failed: %s — %s\n", issue.Key, err)
				fail++
			} else {
				success++
			}
		}

		fmt.Fprintf(os.Stderr, "done: %d moved, %d failed/skipped\n", success, fail)
		return nil
	},
}
