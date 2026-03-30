package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var openURLOnly bool

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().BoolVar(&openURLOnly, "url", false, "Print URL only, don't open browser")
}

var openCmd = &cobra.Command{
	Use:   "open <issue-key>",
	Short: "Open an issue in the browser",
	Long: `Open a Jira issue in the default browser, or print its URL.

Examples:
  jx open MLF-5146
  jx open MLF-5146 --url`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		url := buildBrowseURL(c.Server(), args[0])

		if openURLOnly {
			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		}

		var openCmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			openCmd = exec.Command("xdg-open", url)
		case "darwin":
			openCmd = exec.Command("open", url)
		default:
			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		}

		if err := openCmd.Start(); err != nil {
			// Fallback: print URL if browser can't open
			fmt.Fprintln(cmd.OutOrStdout(), url)
		}
		return nil
	},
}
