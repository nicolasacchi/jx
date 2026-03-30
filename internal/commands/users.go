package commands

import (
	"context"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var usersQuery string

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersMeCmd)

	usersListCmd.Flags().StringVar(&usersQuery, "query", "", "Search by name or email")
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage Jira users",
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "Search users",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"maxResults": {strconv.Itoa(limitFlag)}}
		if usersQuery != "" {
			params.Set("query", usersQuery)
		}
		data, err := c.Get(context.Background(), "rest/api/3/user/search", params)
		if err != nil {
			return err
		}
		return printData("users.list", data)
	},
}

var usersMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current user",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/myself", nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}
