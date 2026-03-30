package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	sprintBoard     int
	sprintName      string
	sprintStartDate string
	sprintEndDate   string
)

func init() {
	rootCmd.AddCommand(sprintsCmd)
	sprintsCmd.AddCommand(sprintsListCmd)
	sprintsCmd.AddCommand(sprintsActiveCmd)
	sprintsCmd.AddCommand(sprintsGetCmd)
	sprintsCmd.AddCommand(sprintsIssuesCmd)
	sprintsCmd.AddCommand(sprintsCreateCmd)
	sprintsCmd.AddCommand(sprintsStartCmd)
	sprintsCmd.AddCommand(sprintsCloseCmd)

	sprintsListCmd.Flags().IntVar(&sprintBoard, "board", 0, "Board ID (required)")
	sprintsListCmd.MarkFlagRequired("board")

	sprintsActiveCmd.Flags().IntVar(&sprintBoard, "board", 0, "Board ID (required)")
	sprintsActiveCmd.MarkFlagRequired("board")

	sprintsCreateCmd.Flags().IntVar(&sprintBoard, "board", 0, "Board ID (required)")
	sprintsCreateCmd.Flags().StringVar(&sprintName, "name", "", "Sprint name (required)")
	sprintsCreateCmd.Flags().StringVar(&sprintStartDate, "start", "", "Start date (ISO 8601)")
	sprintsCreateCmd.Flags().StringVar(&sprintEndDate, "end", "", "End date (ISO 8601)")
	sprintsCreateCmd.MarkFlagRequired("board")
	sprintsCreateCmd.MarkFlagRequired("name")
}

var sprintsCmd = &cobra.Command{
	Use:   "sprints",
	Short: "Manage sprints",
}

var sprintsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sprints for a board",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"maxResults": {strconv.Itoa(limitFlag)}}
		data, err := c.Get(context.Background(), fmt.Sprintf("rest/agile/1.0/board/%d/sprint", sprintBoard), params)
		if err != nil {
			return err
		}
		return printData("sprints.list", extractValues(data, "values"))
	},
}

var sprintsActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "Show the current active sprint",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"state": {"active"}}
		data, err := c.Get(context.Background(), fmt.Sprintf("rest/agile/1.0/board/%d/sprint", sprintBoard), params)
		if err != nil {
			return err
		}
		return printData("sprints.list", extractValues(data, "values"))
	},
}

var sprintsGetCmd = &cobra.Command{
	Use:   "get <sprint-id>",
	Short: "Get sprint details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/agile/1.0/sprint/"+args[0], nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}

var sprintsIssuesCmd = &cobra.Command{
	Use:   "issues <sprint-id>",
	Short: "List issues in a sprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{
			"maxResults": {strconv.Itoa(limitFlag)},
			"fields":     {defaultSearchFields},
		}
		data, err := c.Get(context.Background(), "rest/agile/1.0/sprint/"+args[0]+"/issue", params)
		if err != nil {
			return err
		}
		return printData("issues.list", flattenIssues(data))
	},
}

// extractValues extracts a named array from a wrapper object.
func extractValues(data json.RawMessage, key string) json.RawMessage {
	var wrapper map[string]json.RawMessage
	if json.Unmarshal(data, &wrapper) == nil {
		if vals, ok := wrapper[key]; ok {
			if total, ok := wrapper["total"]; ok {
				var n int
				json.Unmarshal(total, &n)
				if n > 0 {
					fmt.Fprintf(os.Stderr, "%s: %d total\n", key, n)
				}
			}
			return vals
		}
	}
	return data
}

var sprintsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new sprint",
	Long: `Create a new sprint on a board.

Examples:
  jx sprints create --board 40 --name "Sprint 46"
  jx sprints create --board 40 --name "Sprint 46" --start 2026-04-07T08:00:00Z --end 2026-04-21T08:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{
			"name":          sprintName,
			"originBoardId": sprintBoard,
		}
		if sprintStartDate != "" {
			body["startDate"] = sprintStartDate
		}
		if sprintEndDate != "" {
			body["endDate"] = sprintEndDate
		}
		data, err := c.Post(context.Background(), "rest/agile/1.0/sprint", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "sprint created: %s\n", sprintName)
		}
		return printData("", data)
	},
}

var sprintsStartCmd = &cobra.Command{
	Use:   "start <sprint-id>",
	Short: "Start (activate) a sprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{"state": "active"}
		data, err := c.Put(context.Background(), "rest/agile/1.0/sprint/"+args[0], body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "sprint %s started\n", args[0])
		}
		return printData("", data)
	},
}

var sprintsCloseCmd = &cobra.Command{
	Use:   "close <sprint-id>",
	Short: "Close (complete) a sprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{"state": "closed"}
		data, err := c.Put(context.Background(), "rest/agile/1.0/sprint/"+args[0], body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "sprint %s closed\n", args[0])
		}
		return printData("", data)
	},
}
