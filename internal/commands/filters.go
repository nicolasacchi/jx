package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	filterName string
	filterJQL  string
)

func init() {
	rootCmd.AddCommand(filtersCmd)
	filtersCmd.AddCommand(filtersListCmd)
	filtersCmd.AddCommand(filtersGetCmd)
	filtersCmd.AddCommand(filtersCreateCmd)
	filtersCmd.AddCommand(filtersDeleteCmd)

	filtersCreateCmd.Flags().StringVar(&filterName, "name", "", "Filter name (required)")
	filtersCreateCmd.Flags().StringVar(&filterJQL, "jql", "", "JQL query (required)")
	filtersCreateCmd.MarkFlagRequired("name")
	filtersCreateCmd.MarkFlagRequired("jql")
}

var filtersCmd = &cobra.Command{
	Use:   "filters",
	Short: "Manage saved JQL filters",
}

var filtersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your saved filters",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/filter/my", nil)
		if err != nil {
			return err
		}

		var filters []json.RawMessage
		if json.Unmarshal(data, &filters) == nil {
			fmt.Fprintf(os.Stderr, "filters: %d\n", len(filters))
			var rows []map[string]any
			for _, raw := range filters {
				var f struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Owner struct{ DisplayName string } `json:"owner"`
					JQL   string `json:"jql"`
				}
				if json.Unmarshal(raw, &f) != nil {
					continue
				}
				rows = append(rows, map[string]any{
					"id":    f.ID,
					"name":  f.Name,
					"owner": f.Owner.DisplayName,
					"jql":   f.JQL,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("filters.list", out)
		}
		return printData("", data)
	},
}

var filtersGetCmd = &cobra.Command{
	Use:   "get <filter-id>",
	Short: "Get a saved filter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/filter/"+args[0], nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}

var filtersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a saved filter",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{
			"name": filterName,
			"jql":  filterJQL,
		}
		data, err := c.Post(context.Background(), "rest/api/3/filter", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			var created struct{ ID string; Name string }
			json.Unmarshal(data, &created)
			fmt.Fprintf(os.Stderr, "filter created: %s (%s)\n", created.Name, created.ID)
		}
		return printData("", data)
	},
}

var filtersDeleteCmd = &cobra.Command{
	Use:   "delete <filter-id>",
	Short: "Delete a saved filter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/filter/"+args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "filter %s deleted\n", args[0])
		}
		return nil
	},
}
