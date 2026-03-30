package commands

import (
	"context"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var boardsProject string

func init() {
	rootCmd.AddCommand(boardsCmd)
	boardsCmd.AddCommand(boardsListCmd)
	boardsCmd.AddCommand(boardsGetCmd)
	boardsCmd.AddCommand(boardsConfigCmd)

	boardsListCmd.Flags().StringVar(&boardsProject, "project", "", "Filter by project key")
}

var boardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "Manage Jira boards",
}

var boardsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List boards",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"maxResults": {strconv.Itoa(limitFlag)}}
		if boardsProject != "" {
			params.Set("projectKeyOrId", boardsProject)
		}
		data, err := c.Get(context.Background(), "rest/agile/1.0/board", params)
		if err != nil {
			return err
		}
		return printData("boards.list", extractValues(data, "values"))
	},
}

var boardsGetCmd = &cobra.Command{
	Use:   "get <board-id>",
	Short: "Get board details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/agile/1.0/board/"+args[0], nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}

var boardsConfigCmd = &cobra.Command{
	Use:   "config <board-id>",
	Short: "Show board configuration (columns, swimlanes)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/agile/1.0/board/"+args[0]+"/configuration", nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}
