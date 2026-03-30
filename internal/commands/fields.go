package commands

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
)

var (
	fieldsCustom bool
	fieldsSearch string
)

func init() {
	rootCmd.AddCommand(fieldsCmd)
	fieldsCmd.AddCommand(fieldsListCmd)

	fieldsListCmd.Flags().BoolVar(&fieldsCustom, "custom", false, "Show only custom fields")
	fieldsListCmd.Flags().StringVar(&fieldsSearch, "search", "", "Search fields by name")
}

var fieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "Discover Jira fields",
}

var fieldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all fields",
	Long: `List Jira fields. Useful for discovering custom field IDs.

Examples:
  jx fields list --custom
  jx fields list --search "story"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/field", nil)
		if err != nil {
			return err
		}

		var fields []map[string]any
		if json.Unmarshal(data, &fields) == nil {
			var filtered []map[string]any
			for _, f := range fields {
				custom, _ := f["custom"].(bool)
				name, _ := f["name"].(string)
				id, _ := f["id"].(string)

				if fieldsCustom && !custom {
					continue
				}
				if fieldsSearch != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(fieldsSearch)) &&
					!strings.Contains(strings.ToLower(id), strings.ToLower(fieldsSearch)) {
					continue
				}

				// Flatten schema type
				schemaType := ""
				if schema, ok := f["schema"].(map[string]any); ok {
					if t, ok := schema["type"].(string); ok {
						schemaType = t
					}
				}

				filtered = append(filtered, map[string]any{
					"id":     id,
					"name":   name,
					"custom": custom,
					"type":   schemaType,
				})
			}
			if filtered == nil {
				filtered = []map[string]any{}
			}
			out, _ := json.Marshal(filtered)
			return printData("fields.list", out)
		}
		return printData("", data)
	},
}
