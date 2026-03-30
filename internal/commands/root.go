package commands

import (
	"errors"
	"os"

	"github.com/nicolasacchi/jx/internal/client"
	"github.com/nicolasacchi/jx/internal/config"
	"github.com/nicolasacchi/jx/internal/output"
	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	emailFlag   string
	tokenFlag   string
	serverFlag  string
	projectFlag string
	jsonFlag    bool
	jqFlag      string
	limitFlag   int
	verboseFlag bool
	quietFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "jx",
	Short: "jx — Jira Explorer CLI for Claude Code",
	Long: `jx is a purpose-built CLI for Jira Cloud, designed for agent integration.
Auto-JSON on pipe, gjson filtering, ADF-native comments, and markdown support.

Usage examples:
  jx issues list --project MLF --limit 10
  jx issues get MLF-5146
  jx search --project MLF --status "In Progress"
  jx comments add MLF-5146 --file context.md
  jx transitions move MLF-5146 "Done"
  jx overview`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// SetVersion sets the version displayed by --version.
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			output.PrintError(apiErr.Error(), apiErr.StatusCode)
			os.Exit(apiErr.ExitCode())
		}
	}
	return err
}

func getClient(cmd *cobra.Command) (*client.Client, error) {
	creds, err := config.LoadCredentials(emailFlag, tokenFlag, serverFlag, projectFlag)
	if err != nil {
		return nil, err
	}
	return client.New(creds.Email, creds.Token, creds.Server, verboseFlag), nil
}

func isJSONMode() bool {
	return output.IsJSON(jsonFlag, jqFlag)
}

func printData(command string, data []byte) error {
	return output.PrintData(command, data, isJSONMode(), jqFlag)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&emailFlag, "email", "", "Jira email (overrides JIRA_EMAIL)")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "Jira API token (overrides JIRA_API_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Jira server URL (overrides JIRA_SERVER)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Use named project from config")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Force JSON output (auto-enabled when piped)")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "Apply gjson path filter to JSON output")
	rootCmd.PersistentFlags().IntVar(&limitFlag, "limit", 50, "Max results")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Print request/response details to stderr")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-error output")
}
