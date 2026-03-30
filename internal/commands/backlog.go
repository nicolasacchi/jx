package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(backlogCmd)
	backlogCmd.AddCommand(backlogMoveCmd)
	backlogCmd.AddCommand(backlogMoveToSprintCmd)
}

var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Manage backlog and sprint issue placement",
}

var backlogMoveCmd = &cobra.Command{
	Use:   "move <issue-keys>",
	Short: "Move issues to the backlog",
	Long: `Move one or more issues to the backlog (remove from sprint).

Examples:
  jx backlog move MLF-5146
  jx backlog move MLF-5146,MLF-5147,MLF-5148`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		keys := strings.Split(args[0], ",")
		body := map[string]any{"issues": keys}
		_, err = c.Post(context.Background(), "rest/agile/1.0/backlog/issue", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "moved to backlog: %s\n", strings.Join(keys, ", "))
		}
		return nil
	},
}

var backlogMoveToSprintCmd = &cobra.Command{
	Use:   "move-to-sprint <sprint-id> <issue-keys>",
	Short: "Move issues into a sprint",
	Long: `Move one or more issues into a specific sprint.

Examples:
  jx backlog move-to-sprint 920 MLF-5146
  jx backlog move-to-sprint 920 MLF-5146,MLF-5147`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		sprintID := args[0]
		keys := strings.Split(args[1], ",")
		body := map[string]any{"issues": keys}
		_, err = c.Post(context.Background(), "rest/agile/1.0/sprint/"+sprintID+"/issue", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "moved to sprint %s: %s\n", sprintID, strings.Join(keys, ", "))
		}
		return nil
	},
}
