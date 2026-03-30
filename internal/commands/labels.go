package commands

import (
	"context"
	"encoding/json"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(labelsCmd)
	labelsCmd.AddCommand(labelsListCmd)
}

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "Manage Jira labels",
}

var labelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/label", nil)
		if err != nil {
			return err
		}

		// Response is {"values": ["label1", "label2", ...], "total": N}
		var resp struct {
			Values []string `json:"values"`
		}
		if json.Unmarshal(data, &resp) == nil {
			var rows []map[string]any
			for _, l := range resp.Values {
				rows = append(rows, map[string]any{"label": l})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("labels.list", out)
		}
		return printData("", data)
	},
}
