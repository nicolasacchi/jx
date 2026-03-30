package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nicolasacchi/jx/internal/adf"
	"github.com/spf13/cobra"
)

var (
	commentBody    string
	commentFile    string
	commentADFFile string
)

func init() {
	rootCmd.AddCommand(commentsCmd)
	commentsCmd.AddCommand(commentsListCmd)
	commentsCmd.AddCommand(commentsAddCmd)
	commentsCmd.AddCommand(commentsEditCmd)
	commentsCmd.AddCommand(commentsDeleteCmd)

	commentsAddCmd.Flags().StringVar(&commentBody, "body", "", "Comment text (plain text)")
	commentsAddCmd.Flags().StringVar(&commentFile, "file", "", "Markdown file for comment (converted to ADF)")
	commentsAddCmd.Flags().StringVar(&commentADFFile, "adf-file", "", "Raw ADF JSON file for comment")

	commentsEditCmd.Flags().StringVar(&commentBody, "body", "", "Updated comment text")
	commentsEditCmd.Flags().StringVar(&commentFile, "file", "", "Markdown file for comment")
}

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Manage issue comments (ADF-native)",
}

var commentsListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List comments on an issue",
	Long: `List all comments on a Jira issue.

Examples:
  jx comments list MLF-5146
  jx comments list MLF-5146 --jq '#.{id:id,author:author}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/comment", nil)
		if err != nil {
			return err
		}

		return printData("comments.list", flattenComments(data))
	},
}

var commentsAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Add a comment to an issue",
	Long: `Add a comment with full ADF formatting support.

--body: Simple text (wrapped in a paragraph)
--file: Markdown file, automatically converted to ADF (code blocks, headings, lists)
--adf-file: Raw ADF JSON file for full control

Examples:
  jx comments add MLF-5146 --body "Deployed in PR #6966"
  jx comments add MLF-5146 --file context.md
  jx comments add MLF-5146 --adf-file comment.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		var adfBody any
		switch {
		case commentFile != "":
			content, err := os.ReadFile(commentFile)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			adfBody = adf.FromMarkdown(string(content))

		case commentADFFile != "":
			content, err := os.ReadFile(commentADFFile)
			if err != nil {
				return fmt.Errorf("read ADF file: %w", err)
			}
			var raw json.RawMessage
			if err := json.Unmarshal(content, &raw); err != nil {
				return fmt.Errorf("invalid ADF JSON: %w", err)
			}
			adfBody = raw

		case commentBody != "":
			adfBody = adf.New().Paragraph(commentBody).Build()

		default:
			return fmt.Errorf("provide --body, --file, or --adf-file")
		}

		body := map[string]any{"body": adfBody}
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
				fmt.Fprintf(os.Stderr, "comment added: %s\n", created.ID)
			}
		}

		return printData("", data)
	},
}

var commentsEditCmd = &cobra.Command{
	Use:   "edit <comment-id> --issue <issue-key>",
	Short: "Edit a comment",
	Long: `Edit an existing comment. Requires the issue key and comment ID.

Examples:
  jx comments edit 12345 --issue MLF-5146 --body "Updated text"
  jx comments edit 12345 --issue MLF-5146 --file updated.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		issue, _ := cmd.Flags().GetString("issue")
		if issue == "" {
			return fmt.Errorf("--issue flag required")
		}

		var adfBody any
		switch {
		case commentFile != "":
			content, err := os.ReadFile(commentFile)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			adfBody = adf.FromMarkdown(string(content))
		case commentBody != "":
			adfBody = adf.New().Paragraph(commentBody).Build()
		default:
			return fmt.Errorf("provide --body or --file")
		}

		body := map[string]any{"body": adfBody}
		_, err = c.Put(context.Background(), "rest/api/3/issue/"+issue+"/comment/"+args[0], body)
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "comment updated: %s\n", args[0])
		}
		return nil
	},
}

var commentsDeleteCmd = &cobra.Command{
	Use:   "delete <comment-id> --issue <issue-key>",
	Short: "Delete a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		issue, _ := cmd.Flags().GetString("issue")
		if issue == "" {
			return fmt.Errorf("--issue flag required")
		}

		if err := c.Delete(context.Background(), "rest/api/3/issue/"+issue+"/comment/"+args[0]); err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "comment deleted: %s\n", args[0])
		}
		return nil
	},
}

func init() {
	commentsEditCmd.Flags().String("issue", "", "Issue key (required)")
	commentsEditCmd.MarkFlagRequired("issue")
	commentsDeleteCmd.Flags().String("issue", "", "Issue key (required)")
	commentsDeleteCmd.MarkFlagRequired("issue")
}

// flattenComments converts the Jira comment response into a flat list.
func flattenComments(data json.RawMessage) json.RawMessage {
	var resp struct {
		Comments []json.RawMessage `json:"comments"`
		Total    int               `json:"total"`
	}

	if json.Unmarshal(data, &resp) == nil && resp.Comments != nil {
		if resp.Total > 0 {
			fmt.Fprintf(os.Stderr, "comments: %d total\n", resp.Total)
		}
		var flattened []map[string]any
		for _, raw := range resp.Comments {
			var comment struct {
				ID     string `json:"id"`
				Author struct {
					DisplayName string `json:"displayName"`
				} `json:"author"`
				Created string          `json:"created"`
				Updated string          `json:"updated"`
				Body    json.RawMessage `json:"body"`
			}
			if json.Unmarshal(raw, &comment) != nil {
				continue
			}
			flat := map[string]any{
				"id":      comment.ID,
				"author":  comment.Author.DisplayName,
				"created": comment.Created,
				"updated": comment.Updated,
			}
			// Extract plain text from ADF body for table display
			flat["body"] = extractADFText(comment.Body)
			flattened = append(flattened, flat)
		}
		if flattened == nil {
			flattened = []map[string]any{}
		}
		out, _ := json.Marshal(flattened)
		return out
	}
	return data
}

// extractADFText extracts plain text from an ADF document for display.
func extractADFText(data json.RawMessage) string {
	var doc struct {
		Content []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return ""
	}
	var parts []string
	for _, block := range doc.Content {
		for _, inline := range block.Content {
			if inline.Text != "" {
				parts = append(parts, inline.Text)
			}
		}
	}
	return strings.Join(parts, " ")
}
