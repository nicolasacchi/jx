package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nicolasacchi/jx/internal/config"
	"github.com/spf13/cobra"
)

var (
	configEmail  string
	configToken  string
	configServer string
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configCurrentCmd)

	configAddCmd.Flags().StringVar(&configEmail, "email", "", "Jira account email (required)")
	configAddCmd.Flags().StringVar(&configToken, "token", "", "Jira API token (required)")
	configAddCmd.Flags().StringVar(&configServer, "server", "", "Jira server URL (required)")
	configAddCmd.MarkFlagRequired("email")
	configAddCmd.MarkFlagRequired("token")
	configAddCmd.MarkFlagRequired("server")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage jx configuration",
}

var configAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a named project configuration",
	Long: `Add or update a named project configuration.

Examples:
  jx config add production --email nicola@1000farmacie.it --token ATATT3x... --server https://1000farmacie.atlassian.net`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.AddProject(args[0], configEmail, configToken, configServer); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "Project %q saved\n", args[0])
		}
		return nil
	},
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a named project configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.RemoveProject(args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "Project %q removed\n", args[0])
		}
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ListProjects()
		if err != nil {
			return fmt.Errorf("no config file found; run 'jx config add' first")
		}

		var rows []map[string]any
		for name, p := range cfg.Projects {
			isDefault := "no"
			if name == cfg.DefaultProject {
				isDefault = "yes"
			}
			rows = append(rows, map[string]any{
				"name":    name,
				"email":   p.Email,
				"token":   config.MaskKey(p.Token),
				"server":  p.Server,
				"default": isDefault,
			})
		}

		data, _ := json.Marshal(rows)
		return printData("config.list", data)
	},
}

var configUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SetDefaultProject(args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "Default project set to %q\n", args[0])
		}
		return nil
	},
}

var configCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current default project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ListProjects()
		if err != nil {
			return fmt.Errorf("no config file found")
		}
		if cfg.DefaultProject == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "(none)")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), cfg.DefaultProject)
		return nil
	},
}
