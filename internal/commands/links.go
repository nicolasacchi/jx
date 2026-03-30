package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var linkType string

func init() {
	rootCmd.AddCommand(linksCmd)
	linksCmd.AddCommand(linksCreateCmd)
	linksCmd.AddCommand(linksDeleteCmd)

	linksCreateCmd.Flags().StringVar(&linkType, "type", "Blocks", "Link type (e.g., Blocks, is blocked by, Cloners, Duplicate)")
}

var linksCmd = &cobra.Command{
	Use:   "links",
	Short: "Manage issue links",
}

var linksCreateCmd = &cobra.Command{
	Use:   "create <from-key> <to-key>",
	Short: "Create a link between two issues",
	Long: `Link two issues. Default type is "Blocks".

Examples:
  jx links create MLF-5146 MLF-5145 --type "is blocked by"
  jx links create MLF-5146 MLF-5147 --type "Blocks"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		body := map[string]any{
			"type":         map[string]any{"name": linkType},
			"inwardIssue":  map[string]any{"key": args[0]},
			"outwardIssue": map[string]any{"key": args[1]},
		}

		_, err = c.Post(context.Background(), "rest/api/3/issueLink", body)
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "linked: %s → %s (%s)\n", args[0], args[1], linkType)
		}
		return nil
	},
}

var linksDeleteCmd = &cobra.Command{
	Use:   "delete <link-id>",
	Short: "Delete an issue link",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/issueLink/"+args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "link deleted: %s\n", args[0])
		}
		return nil
	},
}
