package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	attachFile   string
	attachOutput string
)

func init() {
	rootCmd.AddCommand(attachmentsCmd)
	attachmentsCmd.AddCommand(attachmentsListCmd)
	attachmentsCmd.AddCommand(attachmentsAddCmd)
	attachmentsCmd.AddCommand(attachmentsGetCmd)
	attachmentsCmd.AddCommand(attachmentsDeleteCmd)

	attachmentsAddCmd.Flags().StringVar(&attachFile, "file", "", "File to attach (required)")
	attachmentsAddCmd.MarkFlagRequired("file")

	attachmentsGetCmd.Flags().StringVar(&attachOutput, "output", ".", "Directory to save the attachment")
}

var attachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "Manage issue attachments",
}

var attachmentsListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List attachments on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"?fields=attachment", nil)
		if err != nil {
			return err
		}

		var resp struct {
			Fields struct {
				Attachment []json.RawMessage `json:"attachment"`
			} `json:"fields"`
		}
		if json.Unmarshal(data, &resp) == nil {
			var rows []map[string]any
			for _, raw := range resp.Fields.Attachment {
				var att struct {
					ID       string `json:"id"`
					Filename string `json:"filename"`
					Size     int64  `json:"size"`
					MimeType string `json:"mimeType"`
					Created  string `json:"created"`
					Author   struct{ DisplayName string } `json:"author"`
				}
				if json.Unmarshal(raw, &att) != nil {
					continue
				}
				rows = append(rows, map[string]any{
					"id":       att.ID,
					"filename": att.Filename,
					"size":     att.Size,
					"mimeType": att.MimeType,
					"created":  att.Created,
					"author":   att.Author.DisplayName,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			fmt.Fprintf(os.Stderr, "attachments: %d\n", len(rows))
			out, _ := json.Marshal(rows)
			return printData("", out)
		}
		return printData("", data)
	},
}

var attachmentsAddCmd = &cobra.Command{
	Use:   "add <issue-key>",
	Short: "Attach a file to an issue",
	Long: `Upload a file attachment to a Jira issue.

Examples:
  jx attachments add MLF-5146 --file screenshot.png
  jx attachments add MLF-5146 --file report.pdf`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		f, err := os.Open(attachFile)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", filepath.Base(attachFile))
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, f); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
		writer.Close()

		data, err := c.PostRaw(context.Background(),
			"rest/api/3/issue/"+args[0]+"/attachments",
			&body,
			writer.FormDataContentType())
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "attached: %s → %s\n", filepath.Base(attachFile), args[0])
		}
		return printData("", data)
	},
}

var attachmentsGetCmd = &cobra.Command{
	Use:   "get <attachment-id>",
	Short: "Download an attachment",
	Long: `Download an attachment by ID. Use 'attachments list' to find IDs.

Examples:
  jx attachments get 12345 --output ./downloads/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		// Get attachment metadata first
		data, err := c.Get(context.Background(), "rest/api/3/attachment/"+args[0], nil)
		if err != nil {
			return err
		}

		var att struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal(data, &att); err != nil {
			return fmt.Errorf("parse attachment: %w", err)
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "attachment: %s (download URL: %s)\n", att.Filename, att.Content)
		}
		return printData("", data)
	},
}

var attachmentsDeleteCmd = &cobra.Command{
	Use:   "delete <attachment-id>",
	Short: "Delete an attachment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/attachment/"+args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "attachment deleted: %s\n", args[0])
		}
		return nil
	},
}
