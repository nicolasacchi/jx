package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	worklogTime    string
	worklogComment string
)

func init() {
	rootCmd.AddCommand(worklogsCmd)
	worklogsCmd.AddCommand(worklogsListCmd)
	worklogsCmd.AddCommand(worklogsAddCmd)

	worklogsAddCmd.Flags().StringVar(&worklogTime, "time", "", "Time spent (e.g., 2h, 30m, 1d)")
	worklogsAddCmd.Flags().StringVar(&worklogComment, "comment", "", "Work description")
	worklogsAddCmd.MarkFlagRequired("time")
}

var worklogsCmd = &cobra.Command{
	Use:   "worklogs",
	Short: "Manage issue worklogs",
}

var worklogsListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List worklogs on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/worklog", nil)
		if err != nil {
			return err
		}

		var resp struct {
			Worklogs []json.RawMessage `json:"worklogs"`
			Total    int               `json:"total"`
		}
		if json.Unmarshal(data, &resp) == nil && resp.Worklogs != nil {
			if resp.Total > 0 {
				fmt.Fprintf(os.Stderr, "worklogs: %d total\n", resp.Total)
			}
			var flattened []map[string]any
			for _, raw := range resp.Worklogs {
				var wl struct {
					ID        string `json:"id"`
					Author    struct{ DisplayName string } `json:"author"`
					Created   string `json:"created"`
					Updated   string `json:"updated"`
					TimeSpent string `json:"timeSpent"`
					Comment   json.RawMessage `json:"comment"`
				}
				if json.Unmarshal(raw, &wl) != nil {
					continue
				}
				flattened = append(flattened, map[string]any{
					"id":         wl.ID,
					"author":     wl.Author.DisplayName,
					"timeSpent":  wl.TimeSpent,
					"created":    wl.Created,
					"comment":    extractADFText(wl.Comment),
				})
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

var worklogsAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Log time on an issue",
	Long: `Add a worklog entry to an issue.

Examples:
  jx worklogs add MLF-5146 --time 2h --comment "Code review"
  jx worklogs add MLF-5146 --time 30m`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		body := map[string]any{
			"timeSpent": worklogTime,
		}
		if worklogComment != "" {
			body["comment"] = map[string]any{
				"version": 1,
				"type":    "doc",
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": worklogComment}},
					},
				},
			}
		}

		data, err := c.Post(context.Background(), "rest/api/3/issue/"+args[0]+"/worklog", body)
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "logged: %s on %s\n", worklogTime, args[0])
		}
		return printData("", data)
	},
}
