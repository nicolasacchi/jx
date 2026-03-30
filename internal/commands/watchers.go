package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var watcherUser string

func init() {
	rootCmd.AddCommand(watchersCmd)
	watchersCmd.AddCommand(watchersListCmd)
	watchersCmd.AddCommand(watchersAddCmd)
	watchersCmd.AddCommand(watchersRemoveCmd)

	watchersAddCmd.Flags().StringVar(&watcherUser, "user", "", "Account ID to add as watcher")
	watchersAddCmd.MarkFlagRequired("user")

	watchersRemoveCmd.Flags().StringVar(&watcherUser, "user", "", "Account ID to remove")
	watchersRemoveCmd.MarkFlagRequired("user")
}

var watchersCmd = &cobra.Command{
	Use:   "watchers",
	Short: "Manage issue watchers",
}

var watchersListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List watchers on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/watchers", nil)
		if err != nil {
			return err
		}

		var resp struct {
			WatchCount int `json:"watchCount"`
			Watchers   []struct {
				AccountID   string `json:"accountId"`
				DisplayName string `json:"displayName"`
				Active      bool   `json:"active"`
			} `json:"watchers"`
		}
		if json.Unmarshal(data, &resp) == nil {
			fmt.Fprintf(os.Stderr, "watchers: %d\n", resp.WatchCount)
			var rows []map[string]any
			for _, w := range resp.Watchers {
				rows = append(rows, map[string]any{
					"accountId":   w.AccountID,
					"displayName": w.DisplayName,
					"active":      w.Active,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("", out)
		}
		return printData("", data)
	},
}

var watchersAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Add a watcher",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		// Jira expects the account ID as a quoted JSON string body
		_, err = c.Post(context.Background(), "rest/api/3/issue/"+args[0]+"/watchers", watcherUser)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "watcher added: %s → %s\n", watcherUser, args[0])
		}
		return nil
	},
}

var watchersRemoveCmd = &cobra.Command{
	Use:   "remove <issue-key>",
	Short: "Remove a watcher",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/issue/"+args[0]+"/watchers?accountId="+watcherUser); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "watcher removed: %s from %s\n", watcherUser, args[0])
		}
		return nil
	},
}
