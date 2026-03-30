package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	propKey   string
	propValue string
)

func init() {
	rootCmd.AddCommand(propertiesCmd)
	propertiesCmd.AddCommand(propertiesListCmd)
	propertiesCmd.AddCommand(propertiesGetCmd)
	propertiesCmd.AddCommand(propertiesSetCmd)
	propertiesCmd.AddCommand(propertiesDeleteCmd)

	propertiesGetCmd.Flags().StringVar(&propKey, "key", "", "Property key (required)")
	propertiesGetCmd.MarkFlagRequired("key")

	propertiesSetCmd.Flags().StringVar(&propKey, "key", "", "Property key (required)")
	propertiesSetCmd.Flags().StringVar(&propValue, "value", "", "Property value as JSON (required)")
	propertiesSetCmd.MarkFlagRequired("key")
	propertiesSetCmd.MarkFlagRequired("value")

	propertiesDeleteCmd.Flags().StringVar(&propKey, "key", "", "Property key (required)")
	propertiesDeleteCmd.MarkFlagRequired("key")
}

var propertiesCmd = &cobra.Command{
	Use:     "properties",
	Aliases: []string{"props"},
	Short:   "Manage issue custom properties/metadata",
}

var propertiesListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List properties on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/properties", nil)
		if err != nil {
			return err
		}

		var resp struct {
			Keys []struct {
				Key string `json:"key"`
			} `json:"keys"`
		}
		if json.Unmarshal(data, &resp) == nil {
			fmt.Fprintf(os.Stderr, "properties: %d\n", len(resp.Keys))
			var rows []map[string]any
			for _, k := range resp.Keys {
				rows = append(rows, map[string]any{"key": k.Key})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("properties.list", out)
		}
		return printData("", data)
	},
}

var propertiesGetCmd = &cobra.Command{
	Use:   "get <issue-key>",
	Short: "Get a property value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0]+"/properties/"+propKey, nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}

var propertiesSetCmd = &cobra.Command{
	Use:   "set <issue-key>",
	Short: "Set a property value",
	Long: `Set a custom property on an issue. Value must be valid JSON.

Examples:
  jx properties set MLF-5146 --key "deploy.status" --value '{"env":"production","sha":"abc123"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		var value any
		if err := json.Unmarshal([]byte(propValue), &value); err != nil {
			return fmt.Errorf("invalid JSON value: %w", err)
		}

		_, err = c.Put(context.Background(), "rest/api/3/issue/"+args[0]+"/properties/"+propKey, value)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "property set: %s.%s\n", args[0], propKey)
		}
		return nil
	},
}

var propertiesDeleteCmd = &cobra.Command{
	Use:   "delete <issue-key>",
	Short: "Delete a property",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/issue/"+args[0]+"/properties/"+propKey); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "property deleted: %s.%s\n", args[0], propKey)
		}
		return nil
	},
}
