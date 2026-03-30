package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nicolasacchi/jx/internal/adf"
	"github.com/spf13/cobra"
)

var (
	contextFile string
	contextBody string
)

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextAddCmd)

	contextAddCmd.Flags().StringVar(&contextFile, "file", "", "Markdown file containing the context prompt")
	contextAddCmd.Flags().StringVar(&contextBody, "body", "", "Inline context text")
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Cross-instance context sharing via Jira comments",
}

var contextAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Post a structured context comment for cross-instance handoff",
	Long: `Post a comment designed for another Claude Code instance to consume.

The comment has two parts:
1. An italic intro: "Copy the prompt below into Claude Code for full task context:"
2. A code block containing the full context prompt

The code block is properly formatted as ADF — no more broken code blocks.

Examples:
  jx context add MLF-5146 --file context.md
  jx context add MLF-5146 --body "## Task\nModify controller..."`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		var promptText string
		switch {
		case contextFile != "":
			content, err := os.ReadFile(contextFile)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			promptText = string(content)
		case contextBody != "":
			promptText = contextBody
		default:
			return fmt.Errorf("provide --file or --body")
		}

		// Build the ADF comment: intro paragraph + code block with the prompt
		doc := adf.New().
			ParagraphWithMarks([]adf.Node{
				adf.Italic("Copy the prompt below into Claude Code for full task context:"),
			}).
			CodeBlock("text", promptText).
			Build()

		body := map[string]any{"body": doc}
		data, err := c.Post(context.Background(), "rest/api/3/issue/"+args[0]+"/comment", body)
		if err != nil {
			return err
		}

		var created struct {
			ID   string `json:"id"`
			Self string `json:"self"`
		}
		if json.Unmarshal(data, &created) == nil && created.ID != "" {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "context comment posted: %s (%s)\n", created.ID, buildBrowseURL(c.Server(), args[0]))
			}
		}

		return printData("", data)
	},
}
