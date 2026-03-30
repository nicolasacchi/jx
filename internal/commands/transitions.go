package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nicolasacchi/jx/internal/adf"
	"github.com/spf13/cobra"
)

var transitionComment string

func init() {
	rootCmd.AddCommand(transitionsCmd)
	transitionsCmd.AddCommand(transitionsListCmd)
	transitionsCmd.AddCommand(transitionsMoveCmd)

	transitionsMoveCmd.Flags().StringVar(&transitionComment, "comment", "", "Add a comment with the transition")
}

var transitionsCmd = &cobra.Command{
	Use:   "transitions",
	Short: "Manage issue workflow transitions",
}

var transitionsListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List available transitions for an issue",
	Long: `Show which status transitions are available for an issue.

Examples:
  jx transitions list MLF-5146`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/transitions", nil)
		if err != nil {
			return err
		}

		var resp struct {
			Transitions []json.RawMessage `json:"transitions"`
		}
		if json.Unmarshal(data, &resp) == nil && resp.Transitions != nil {
			var flattened []map[string]any
			for _, raw := range resp.Transitions {
				var t struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					To   struct {
						Name string `json:"name"`
					} `json:"to"`
				}
				if json.Unmarshal(raw, &t) != nil {
					continue
				}
				flattened = append(flattened, map[string]any{
					"id":        t.ID,
					"name":      t.Name,
					"to_status": t.To.Name,
				})
			}
			if flattened == nil {
				flattened = []map[string]any{}
			}
			out, _ := json.Marshal(flattened)
			return printData("transitions.list", out)
		}
		return printData("", data)
	},
}

var transitionsMoveCmd = &cobra.Command{
	Use:   "move <issue-key> <status>",
	Short: "Transition an issue to a new status",
	Long: `Move an issue to a new workflow status. The status name is matched
case-insensitively against available transitions.

Examples:
  jx transitions move MLF-5146 "In Progress"
  jx transitions move MLF-5146 Done --comment "Completed in PR #6966"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		issueKey := args[0]
		targetStatus := args[1]

		// Get available transitions
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+issueKey+"/transitions", nil)
		if err != nil {
			return err
		}

		var resp struct {
			Transitions []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				To   struct {
					Name string `json:"name"`
				} `json:"to"`
			} `json:"transitions"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parse transitions: %w", err)
		}

		// Find matching transition (case-insensitive)
		var transitionID string
		var matchedName string
		for _, t := range resp.Transitions {
			if strings.EqualFold(t.Name, targetStatus) || strings.EqualFold(t.To.Name, targetStatus) {
				transitionID = t.ID
				matchedName = t.To.Name
				break
			}
		}
		if transitionID == "" {
			available := make([]string, len(resp.Transitions))
			for i, t := range resp.Transitions {
				available[i] = fmt.Sprintf("%s → %s", t.Name, t.To.Name)
			}
			return fmt.Errorf("no transition matching %q; available: %s", targetStatus, strings.Join(available, ", "))
		}

		body := map[string]any{
			"transition": map[string]any{"id": transitionID},
		}

		if transitionComment != "" {
			body["update"] = map[string]any{
				"comment": []any{
					map[string]any{
						"add": map[string]any{
							"body": adf.New().Paragraph(transitionComment).Build(),
						},
					},
				},
			}
		}

		_, err = c.Post(context.Background(), "rest/api/3/issue/"+issueKey+"/transitions", body)
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "moved: %s → %s\n", issueKey, matchedName)
		}
		return nil
	},
}
