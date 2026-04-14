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
	attachmentsCmd.AddCommand(attachmentsDownloadAllCmd)
	attachmentsCmd.AddCommand(attachmentsDeleteCmd)

	attachmentsAddCmd.Flags().StringVar(&attachFile, "file", "", "File to attach (required)")
	attachmentsAddCmd.MarkFlagRequired("file")

	attachmentsGetCmd.Flags().StringVar(&attachOutput, "output", ".", "Directory to save the attachment")
	attachmentsDownloadAllCmd.Flags().StringVar(&attachOutput, "output", ".", "Directory to save attachments (per-issue subdir created)")
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
	Short: "Download an attachment by ID",
	Long: `Download an attachment to disk by its ID. Use 'attachments list' to find IDs.

Examples:
  jx attachments get 12345 --output ./downloads/

Reads the attachment metadata to learn the filename, then downloads the binary
content. Skips the download if the destination file already exists with non-zero
size (idempotent re-runs). Prints the absolute path to stderr on success.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		// 1. Fetch metadata to learn the filename + verify the attachment exists.
		meta, err := c.Get(context.Background(), "rest/api/3/attachment/"+args[0], nil)
		if err != nil {
			return err
		}
		var att struct {
			Filename string `json:"filename"`
			Size     int64  `json:"size"`
		}
		if err := json.Unmarshal(meta, &att); err != nil {
			return fmt.Errorf("parse attachment metadata: %w", err)
		}
		if att.Filename == "" {
			return fmt.Errorf("attachment %s has no filename in metadata", args[0])
		}

		// 2. Ensure output dir exists.
		if err := os.MkdirAll(attachOutput, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		outPath := filepath.Join(attachOutput, att.Filename)

		// 3. Idempotent skip: if destination exists with a non-zero size, assume done.
		if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "skipped (already exists): %s (%d bytes)\n", outPath, info.Size())
			}
			return nil
		}

		// 4. Download the binary content. Uses GetBinary (5min timeout, no JSON wrapping).
		body, err := c.GetBinary(context.Background(), "rest/api/3/attachment/content/"+args[0], nil)
		if err != nil {
			return fmt.Errorf("download content: %w", err)
		}

		// 5. Write to disk.
		if err := os.WriteFile(outPath, body, 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "downloaded: %s (%d bytes)\n", outPath, len(body))
		}
		return nil
	},
}

var attachmentsDownloadAllCmd = &cobra.Command{
	Use:   "download-all <issue-key>",
	Short: "Download all attachments for an issue",
	Long: `Download every attachment from a Jira issue into a per-issue subdirectory
under --output. Skips files already present (idempotent).

Examples:
  jx attachments download-all MLF-5146 --output ./attachments/
  # Creates ./attachments/MLF-5146/<filename> for each attachment

Prints a one-line-per-file progress on stderr and a summary at the end.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		issueKey := args[0]

		// 1. Fetch the issue with only the attachment field for compactness.
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+issueKey+"?fields=attachment", nil)
		if err != nil {
			return err
		}
		var resp struct {
			Fields struct {
				Attachment []struct {
					ID       string `json:"id"`
					Filename string `json:"filename"`
					Content  string `json:"content"`
					Size     int64  `json:"size"`
				} `json:"attachment"`
			} `json:"fields"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parse issue attachments: %w", err)
		}

		// 2. Ensure per-issue subdir exists.
		issueDir := filepath.Join(attachOutput, issueKey)
		if err := os.MkdirAll(issueDir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		total := len(resp.Fields.Attachment)
		if total == 0 {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "no attachments on %s\n", issueKey)
			}
			return nil
		}

		downloaded := 0
		skipped := 0
		failed := 0
		for i, att := range resp.Fields.Attachment {
			outPath := filepath.Join(issueDir, att.Filename)

			// Idempotent skip
			if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
				skipped++
				if !quietFlag {
					fmt.Fprintf(os.Stderr, "  [%d/%d] %s (skipped, exists)\n", i+1, total, att.Filename)
				}
				continue
			}

			// Download via the content URL extracted from metadata. Note: we use the
			// REST /attachment/content/{id} endpoint via GetBinary, not the absolute URL,
			// to ensure auth headers are applied correctly.
			body, derr := c.GetBinary(context.Background(), "rest/api/3/attachment/content/"+att.ID, nil)
			if derr != nil {
				failed++
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s FAILED: %v\n", i+1, total, att.Filename, derr)
				continue
			}
			if err := os.WriteFile(outPath, body, 0o644); err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s FAILED to write: %v\n", i+1, total, att.Filename, err)
				continue
			}
			downloaded++
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s (%d KB)\n", i+1, total, att.Filename, len(body)/1024)
			}
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "%s: %d downloaded, %d skipped, %d failed (total %d) → %s\n",
				issueKey, downloaded, skipped, failed, total, issueDir)
		}
		if failed > 0 {
			return fmt.Errorf("%d of %d attachments failed", failed, total)
		}
		return nil
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
