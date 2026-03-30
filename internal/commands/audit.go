package commands

import (
	"context"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	auditFrom   string
	auditTo     string
	auditFilter string
)

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.AddCommand(auditListCmd)

	auditListCmd.Flags().StringVar(&auditFrom, "from", "", "Start date (ISO 8601, e.g., 2026-03-23)")
	auditListCmd.Flags().StringVar(&auditTo, "to", "", "End date (ISO 8601)")
	auditListCmd.Flags().StringVar(&auditFilter, "filter", "", "Filter string")
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Jira audit log",
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit log entries",
	Long: `Fetch Jira audit log records. Requires admin permissions.

Examples:
  jx audit list --from 2026-03-23
  jx audit list --filter "issue.updated" --limit 20`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{"limit": {strconv.Itoa(limitFlag)}}
		if auditFrom != "" {
			params.Set("from", auditFrom)
		}
		if auditTo != "" {
			params.Set("to", auditTo)
		}
		if auditFilter != "" {
			params.Set("filter", auditFilter)
		}

		data, err := c.Get(context.Background(), "rest/api/3/auditing/record", params)
		if err != nil {
			return err
		}

		return printData("", extractValues(data, "records"))
	},
}
