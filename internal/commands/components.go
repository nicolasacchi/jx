package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	componentsProject string
	componentName     string
	componentLead     string
)

func init() {
	rootCmd.AddCommand(componentsCmd)
	componentsCmd.AddCommand(componentsListCmd)
	componentsCmd.AddCommand(componentsCreateCmd)
	componentsCmd.AddCommand(componentsDeleteCmd)

	componentsListCmd.Flags().StringVar(&componentsProject, "project", "", "Project key (required)")
	componentsListCmd.MarkFlagRequired("project")

	componentsCreateCmd.Flags().StringVar(&componentsProject, "project", "", "Project key (required)")
	componentsCreateCmd.Flags().StringVar(&componentName, "name", "", "Component name (required)")
	componentsCreateCmd.Flags().StringVar(&componentLead, "lead", "", "Lead account ID")
	componentsCreateCmd.MarkFlagRequired("project")
	componentsCreateCmd.MarkFlagRequired("name")
}

var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Manage project components",
}

var componentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List components for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/project/"+componentsProject+"/components", nil)
		if err != nil {
			return err
		}

		var items []json.RawMessage
		if json.Unmarshal(data, &items) == nil {
			var rows []map[string]any
			for _, raw := range items {
				var comp struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Lead *struct{ DisplayName string } `json:"lead"`
				}
				if json.Unmarshal(raw, &comp) != nil {
					continue
				}
				row := map[string]any{"id": comp.ID, "name": comp.Name}
				if comp.Lead != nil {
					row["lead"] = comp.Lead.DisplayName
				}
				rows = append(rows, row)
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			fmt.Fprintf(os.Stderr, "components: %d\n", len(rows))
			out, _ := json.Marshal(rows)
			return printData("components.list", out)
		}
		return printData("", data)
	},
}

var componentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a component",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{
			"name":    componentName,
			"project": componentsProject,
		}
		if componentLead != "" {
			body["leadAccountId"] = componentLead
		}
		data, err := c.Post(context.Background(), "rest/api/3/component", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "component created: %s\n", componentName)
		}
		return printData("", data)
	},
}

var componentsDeleteCmd = &cobra.Command{
	Use:   "delete <component-id>",
	Short: "Delete a component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/component/"+args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "component %s deleted\n", args[0])
		}
		return nil
	},
}
