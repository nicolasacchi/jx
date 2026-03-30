package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	remoteLinkURL   string
	remoteLinkTitle string
)

func init() {
	rootCmd.AddCommand(remoteLinksCmd)
	remoteLinksCmd.AddCommand(remoteLinksListCmd)
	remoteLinksCmd.AddCommand(remoteLinksCreateCmd)
	remoteLinksCmd.AddCommand(remoteLinksDeleteCmd)

	remoteLinksCreateCmd.Flags().StringVar(&remoteLinkURL, "url", "", "External URL (required)")
	remoteLinksCreateCmd.Flags().StringVar(&remoteLinkTitle, "title", "", "Link title (required)")
	remoteLinksCreateCmd.MarkFlagRequired("url")
	remoteLinksCreateCmd.MarkFlagRequired("title")
}

var remoteLinksCmd = &cobra.Command{
	Use:     "remote-links",
	Aliases: []string{"rl"},
	Short:   "Manage external links on issues",
}

var remoteLinksListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List remote links on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/remotelink", nil)
		if err != nil {
			return err
		}

		var links []json.RawMessage
		if json.Unmarshal(data, &links) == nil {
			fmt.Fprintf(os.Stderr, "remote links: %d\n", len(links))
			var rows []map[string]any
			for _, raw := range links {
				var rl struct {
					ID     int `json:"id"`
					Object struct {
						URL   string `json:"url"`
						Title string `json:"title"`
					} `json:"object"`
				}
				if json.Unmarshal(raw, &rl) != nil {
					continue
				}
				rows = append(rows, map[string]any{
					"id":    rl.ID,
					"title": rl.Object.Title,
					"url":   rl.Object.URL,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("remote-links.list", out)
		}
		return printData("", data)
	},
}

var remoteLinksCreateCmd = &cobra.Command{
	Use:   "create <issue-key>",
	Short: "Create a remote link",
	Long: `Link an issue to an external URL (e.g., GitHub PR, deployment).

Examples:
  jx remote-links create MLF-5146 --url "https://github.com/.../pull/6966" --title "PR #6966"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{
			"object": map[string]any{
				"url":   remoteLinkURL,
				"title": remoteLinkTitle,
			},
		}
		data, err := c.Post(context.Background(), "rest/api/3/issue/"+args[0]+"/remotelink", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "remote link created: %s → %s\n", remoteLinkTitle, args[0])
		}
		return printData("", data)
	},
}

var remoteLinksDeleteCmd = &cobra.Command{
	Use:   "delete <issue-key> <link-id>",
	Short: "Delete a remote link",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/issue/"+args[0]+"/remotelink/"+args[1]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "remote link %s deleted from %s\n", args[1], args[0])
		}
		return nil
	},
}
