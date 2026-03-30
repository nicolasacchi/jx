package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	permProject    string
	permPermission string
)

func init() {
	rootCmd.AddCommand(permissionsCmd)
	permissionsCmd.AddCommand(permissionsMineCmd)
	permissionsCmd.AddCommand(permissionsCheckCmd)

	permissionsMineCmd.Flags().StringVar(&permProject, "project", "", "Project key (required)")
	permissionsMineCmd.MarkFlagRequired("project")

	permissionsCheckCmd.Flags().StringVar(&permProject, "project", "", "Project key (required)")
	permissionsCheckCmd.Flags().StringVar(&permPermission, "permission", "", "Permission to check (e.g., EDIT_ISSUES)")
	permissionsCheckCmd.MarkFlagRequired("project")
	permissionsCheckCmd.MarkFlagRequired("permission")
}

// All commonly useful permissions
var allPermissions = []string{
	"BROWSE_PROJECTS",
	"CREATE_ISSUES",
	"EDIT_ISSUES",
	"DELETE_ISSUES",
	"ASSIGN_ISSUES",
	"ASSIGNABLE_USER",
	"CLOSE_ISSUES",
	"TRANSITION_ISSUES",
	"ADD_COMMENTS",
	"DELETE_ALL_COMMENTS",
	"MANAGE_WATCHERS",
	"WORK_ON_ISSUES",
	"MANAGE_SPRINTS_PERMISSION",
	"ADMINISTER_PROJECTS",
}

var permissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Check user permissions",
}

var permissionsMineCmd = &cobra.Command{
	Use:   "mine",
	Short: "Show my permissions for a project",
	Long: `Show which permissions the current user has in a project.

Examples:
  jx permissions mine --project MLF`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{
			"projectKey":  {permProject},
			"permissions": {strings.Join(allPermissions, ",")},
		}
		data, err := c.Get(context.Background(), "rest/api/3/mypermissions", params)
		if err != nil {
			return err
		}

		var resp struct {
			Permissions map[string]struct {
				HavePermission bool `json:"havePermission"`
			} `json:"permissions"`
		}
		if json.Unmarshal(data, &resp) == nil {
			var rows []map[string]any
			for _, perm := range allPermissions {
				if p, ok := resp.Permissions[perm]; ok {
					rows = append(rows, map[string]any{
						"permission": perm,
						"have":       p.HavePermission,
					})
				}
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			fmt.Fprintf(os.Stderr, "permissions for %s: %d checked\n", permProject, len(rows))
			out, _ := json.Marshal(rows)
			return printData("permissions.mine", out)
		}
		return printData("", data)
	},
}

var permissionsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check a specific permission",
	Long: `Check if the current user has a specific permission in a project.

Examples:
  jx permissions check --project MLF --permission EDIT_ISSUES
  jx permissions check --project MLF --permission MANAGE_SPRINTS_PERMISSION`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{
			"projectKey":  {permProject},
			"permissions": {permPermission},
		}
		data, err := c.Get(context.Background(), "rest/api/3/mypermissions", params)
		if err != nil {
			return err
		}

		var resp struct {
			Permissions map[string]struct {
				HavePermission bool `json:"havePermission"`
			} `json:"permissions"`
		}
		if json.Unmarshal(data, &resp) == nil {
			if p, ok := resp.Permissions[permPermission]; ok {
				result := map[string]any{
					"permission": permPermission,
					"project":    permProject,
					"have":       p.HavePermission,
				}
				out, _ := json.Marshal(result)
				return printData("", out)
			}
		}
		return printData("", data)
	},
}
