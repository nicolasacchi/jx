package commands

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/nicolasacchi/jx/internal/jql"
	"github.com/spf13/cobra"
)

var epicsProject string

func init() {
	rootCmd.AddCommand(epicsCmd)
	epicsCmd.AddCommand(epicsListCmd)
	epicsCmd.AddCommand(epicsGetCmd)
	epicsCmd.AddCommand(epicsIssuesCmd)

	epicsListCmd.Flags().StringVar(&epicsProject, "project", "", "Filter by project key")
}

var epicsCmd = &cobra.Command{
	Use:   "epics",
	Short: "Manage epics",
}

var epicsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List epics in a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		q := jql.New().Project(epicsProject).Type("Epic").OrderBy("updated", "DESC")
		params := url.Values{
			"jql":        {q.Build()},
			"maxResults": {strconv.Itoa(limitFlag)},
			"fields":     {defaultSearchFields},
		}
		data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
		if err != nil {
			return err
		}
		return printData("epics.list", flattenIssues(data))
	},
}

var epicsGetCmd = &cobra.Command{
	Use:   "get <epic-key>",
	Short: "Get epic details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "rest/api/3/issue/"+args[0], nil)
		if err != nil {
			return err
		}
		if flat := flattenIssue(data); flat != nil {
			out, _ := json.Marshal(flat)
			return printData("", out)
		}
		return printData("", data)
	},
}

var epicsIssuesCmd = &cobra.Command{
	Use:   "issues <epic-key>",
	Short: "List all issues in an epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		q := jql.New().Epic(args[0]).OrderBy("updated", "DESC")
		params := url.Values{
			"jql":        {q.Build()},
			"maxResults": {strconv.Itoa(limitFlag)},
			"fields":     {defaultSearchFields},
		}
		data, err := c.Get(context.Background(), "rest/api/3/search/jql", params)
		if err != nil {
			return err
		}
		return printData("issues.list", flattenIssues(data))
	},
}
