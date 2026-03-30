package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/nicolasacchi/jx/internal/adf"
	"github.com/nicolasacchi/jx/internal/jql"
	"github.com/spf13/cobra"
)

var (
	sdStatus     string
	sdUpdated    string
	sdType       string
	sdSummary    string
	sdDescFile   string
	sdCreateType string
)

func init() {
	rootCmd.AddCommand(serviceDeskCmd)
	serviceDeskCmd.AddCommand(sdListCmd)
	serviceDeskCmd.AddCommand(sdGetCmd)
	serviceDeskCmd.AddCommand(sdCreateCmd)

	sdListCmd.Flags().StringVar(&sdStatus, "status", "", "Filter by status")
	sdListCmd.Flags().StringVar(&sdUpdated, "updated", "", "Updated since (e.g., -2d)")
	sdListCmd.Flags().StringVar(&sdType, "type", "", "Filter by type (e.g., Bug, Service Request)")

	sdCreateCmd.Flags().StringVar(&sdCreateType, "type", "Service Request", "Issue type")
	sdCreateCmd.Flags().StringVar(&sdSummary, "summary", "", "Summary (required)")
	sdCreateCmd.Flags().StringVar(&sdDescFile, "description-file", "", "Markdown file for description")
	sdCreateCmd.MarkFlagRequired("summary")
}

var serviceDeskCmd = &cobra.Command{
	Use:     "service-desk",
	Aliases: []string{"sd"},
	Short:   "Manage SDD service desk tickets",
}

var sdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SDD service desk tickets",
	Long: `List tickets in the SDD (Service Desk) project.

Examples:
  jx service-desk list --updated -2d
  jx service-desk list --status Open
  jx sd list --type Bug --limit 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		q := jql.New().
			Project("SDD").
			Status(sdStatus).
			Type(sdType).
			UpdatedSince(sdUpdated).
			OrderBy("updated", "DESC")

		params := url.Values{
			"jql":        {q.Build()},
			"maxResults": {strconv.Itoa(limitFlag)},
			"fields":     {defaultSearchFields},
		}

		data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
		if err != nil {
			return err
		}

		return printData("issues.list", flattenIssues(data))
	},
}

var sdGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a service desk ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0], nil)
		if err != nil {
			return err
		}
		if flat := flattenIssue(data); flat != nil {
			out, _ := json.Marshal(flat)
			return printData("issues.get", out)
		}
		return printData("", data)
	},
}

var sdCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a service desk ticket",
	Long: `Create a ticket in the SDD project.

Examples:
  jx service-desk create --summary "App crashes on login"
  jx sd create --type Bug --summary "Payment error" --description-file bug.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		fields := map[string]any{
			"project":   map[string]any{"key": "SDD"},
			"summary":   sdSummary,
			"issuetype": map[string]any{"name": sdCreateType},
		}

		if sdDescFile != "" {
			content, err := os.ReadFile(sdDescFile)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			fields["description"] = adf.FromMarkdown(string(content))
		}

		body := map[string]any{"fields": fields}
		data, err := c.Post(context.Background(), "rest/api/3/issue", body)
		if err != nil {
			return err
		}

		var created struct {
			Key string `json:"key"`
		}
		if json.Unmarshal(data, &created) == nil && created.Key != "" {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "created: %s (%s)\n", created.Key, buildBrowseURL(c.Server(), created.Key))
			}
		}
		return printData("", data)
	},
}
