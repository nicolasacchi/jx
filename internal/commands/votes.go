package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(votesCmd)
	votesCmd.AddCommand(votesListCmd)
	votesCmd.AddCommand(votesAddCmd)
	votesCmd.AddCommand(votesRemoveCmd)
}

var votesCmd = &cobra.Command{
	Use:   "votes",
	Short: "Manage issue votes",
}

var votesListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List voters on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/votes", nil)
		if err != nil {
			return err
		}

		var resp struct {
			Votes  int `json:"votes"`
			Voters []struct {
				AccountID   string `json:"accountId"`
				DisplayName string `json:"displayName"`
			} `json:"voters"`
		}
		if json.Unmarshal(data, &resp) == nil {
			fmt.Fprintf(os.Stderr, "votes: %d\n", resp.Votes)
			var rows []map[string]any
			for _, v := range resp.Voters {
				rows = append(rows, map[string]any{
					"accountId":   v.AccountID,
					"displayName": v.DisplayName,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("votes.list", out)
		}
		return printData("", data)
	},
}

var votesAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Vote for an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		_, err = c.Post(context.Background(), "rest/api/3/issue/"+args[0]+"/votes", nil)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "voted: %s\n", args[0])
		}
		return nil
	},
}

var votesRemoveCmd = &cobra.Command{
	Use:   "remove <issue-key>",
	Short: "Remove your vote from an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/issue/"+args[0]+"/votes"); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "vote removed: %s\n", args[0])
		}
		return nil
	},
}
