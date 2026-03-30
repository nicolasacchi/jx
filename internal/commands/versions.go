package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var (
	versionsProject     string
	versionsName        string
	versionsReleaseDate string
	versionsDesc        string
)

func init() {
	rootCmd.AddCommand(versionsCmd)
	versionsCmd.AddCommand(versionsListCmd)
	versionsCmd.AddCommand(versionsGetCmd)
	versionsCmd.AddCommand(versionsCreateCmd)
	versionsCmd.AddCommand(versionsReleaseCmd)
	versionsCmd.AddCommand(versionsDeleteCmd)

	versionsListCmd.Flags().StringVar(&versionsProject, "project", "", "Project key (required)")
	versionsListCmd.MarkFlagRequired("project")

	versionsCreateCmd.Flags().StringVar(&versionsProject, "project", "", "Project key (required)")
	versionsCreateCmd.Flags().StringVar(&versionsName, "name", "", "Version name (required)")
	versionsCreateCmd.Flags().StringVar(&versionsReleaseDate, "release-date", "", "Release date (YYYY-MM-DD)")
	versionsCreateCmd.Flags().StringVar(&versionsDesc, "description", "", "Description")
	versionsCreateCmd.MarkFlagRequired("project")
	versionsCreateCmd.MarkFlagRequired("name")
}

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Manage project versions/releases",
}

var versionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List versions for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"maxResults": {strconv.Itoa(limitFlag)}}
		data, err := c.Get(context.Background(), "rest/api/3/project/"+versionsProject+"/version", params)
		if err != nil {
			return err
		}

		var resp struct {
			Values []json.RawMessage `json:"values"`
			Total  int               `json:"total"`
		}
		if json.Unmarshal(data, &resp) == nil && resp.Values != nil {
			if resp.Total > 0 {
				fmt.Fprintf(os.Stderr, "versions: %d total\n", resp.Total)
			}
			var rows []map[string]any
			for _, raw := range resp.Values {
				var v struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Released    bool   `json:"released"`
					Archived    bool   `json:"archived"`
					ReleaseDate string `json:"releaseDate"`
					Description string `json:"description"`
				}
				if json.Unmarshal(raw, &v) != nil {
					continue
				}
				status := "unreleased"
				if v.Released {
					status = "released"
				}
				if v.Archived {
					status = "archived"
				}
				rows = append(rows, map[string]any{
					"id":          v.ID,
					"name":        v.Name,
					"status":      status,
					"releaseDate": v.ReleaseDate,
					"description": v.Description,
				})
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			out, _ := json.Marshal(rows)
			return printData("versions.list", out)
		}
		return printData("", data)
	},
}

var versionsGetCmd = &cobra.Command{
	Use:   "get <version-id>",
	Short: "Get version details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/version/"+args[0], nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}

var versionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new version",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		// Need project ID, not key — fetch it first
		projData, err := c.Get(context.Background(), "rest/api/3/project/"+versionsProject, nil)
		if err != nil {
			return err
		}
		var proj struct{ ID string `json:"id"` }
		json.Unmarshal(projData, &proj)

		body := map[string]any{
			"name":      versionsName,
			"projectId": json.Number(proj.ID),
			"released":  false,
			"archived":  false,
		}
		if versionsReleaseDate != "" {
			body["releaseDate"] = versionsReleaseDate
		}
		if versionsDesc != "" {
			body["description"] = versionsDesc
		}

		data, err := c.Post(context.Background(), "rest/api/3/version", body)
		if err != nil {
			return err
		}
		if !quietFlag {
			var created struct{ Name string }
			json.Unmarshal(data, &created)
			fmt.Fprintf(os.Stderr, "created version: %s\n", created.Name)
		}
		return printData("", data)
	},
}

var versionsReleaseCmd = &cobra.Command{
	Use:   "release <version-id>",
	Short: "Mark a version as released",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{
			"released":    true,
			"releaseDate": time.Now().Format("2006-01-02"),
		}
		data, err := c.Put(context.Background(), "rest/api/3/version/"+args[0], body)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "version %s released\n", args[0])
		}
		return printData("", data)
	},
}

var versionsDeleteCmd = &cobra.Command{
	Use:   "delete <version-id>",
	Short: "Delete a version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "rest/api/3/version/"+args[0]); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "version %s deleted\n", args[0])
		}
		return nil
	},
}
