package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// defaultSearchFields is the set of fields requested on search/list operations.
const defaultSearchFields = "summary,status,issuetype,priority,assignee,reporter,created,updated,labels,parent,description,fixVersions,components,resolution,duedate,customfield_10020,customfield_10028,customfield_10014,customfield_10021,issuelinks,subtasks"

// flattenIssue converts a Jira issue response into a flat map for output.
func flattenIssue(raw json.RawMessage) map[string]any {
	var issue struct {
		Key    string `json:"key"`
		ID     string `json:"id"`
		Self   string `json:"self"`
		Fields struct {
			Summary   string `json:"summary"`
			DueDate   string `json:"duedate"`
			Created   string `json:"created"`
			Updated   string `json:"updated"`
			Labels    []string `json:"labels"`
			Status    *struct{ Name string } `json:"status"`
			IssueType *struct{ Name string } `json:"issuetype"`
			Priority  *struct{ Name string } `json:"priority"`
			Assignee  *struct {
				DisplayName string `json:"displayName"`
				AccountID   string `json:"accountId"`
			} `json:"assignee"`
			Reporter *struct {
				DisplayName string `json:"displayName"`
			} `json:"reporter"`
			Parent     *struct{ Key string } `json:"parent"`
			Resolution *struct{ Name string } `json:"resolution"`
			FixVersions []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"fixVersions"`
			Versions []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"versions"`
			Components []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"components"`
			IssueLinks []struct {
				ID   string `json:"id"`
				Type struct {
					Name    string `json:"name"`
					Inward  string `json:"inward"`
					Outward string `json:"outward"`
				} `json:"type"`
				InwardIssue  *struct{ Key string } `json:"inwardIssue"`
				OutwardIssue *struct{ Key string } `json:"outwardIssue"`
			} `json:"issuelinks"`
			Subtasks []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string          `json:"summary"`
					Status  *struct{ Name string } `json:"status"`
				} `json:"fields"`
			} `json:"subtasks"`
			TimeTracking *struct {
				OriginalEstimate  string `json:"originalEstimate"`
				RemainingEstimate string `json:"remainingEstimate"`
				TimeSpent         string `json:"timeSpent"`
			} `json:"timetracking"`
			Votes   *struct{ Votes int } `json:"votes"`
			Watches *struct{ WatchCount int } `json:"watches"`
			Description json.RawMessage `json:"description"`

			// Custom fields
			Sprint     json.RawMessage `json:"customfield_10020"` // Sprint (array)
			StoryPts   *float64        `json:"customfield_10028"` // Story Points
			EpicLink   string          `json:"customfield_10014"` // Epic Link
			Flagged    json.RawMessage `json:"customfield_10021"` // Flagged (array)
		} `json:"fields"`
	}

	if json.Unmarshal(raw, &issue) != nil {
		return nil
	}

	flat := map[string]any{
		"key":     issue.Key,
		"id":      issue.ID,
		"self":    issue.Self,
		"summary": issue.Fields.Summary,
		"created": issue.Fields.Created,
		"updated": issue.Fields.Updated,
		"labels":  issue.Fields.Labels,
	}

	if issue.Fields.Status != nil {
		flat["status"] = issue.Fields.Status.Name
	}
	if issue.Fields.IssueType != nil {
		flat["type"] = issue.Fields.IssueType.Name
	}
	if issue.Fields.Priority != nil {
		flat["priority"] = issue.Fields.Priority.Name
	}
	if issue.Fields.Assignee != nil {
		flat["assignee"] = issue.Fields.Assignee.DisplayName
	}
	if issue.Fields.Reporter != nil {
		flat["reporter"] = issue.Fields.Reporter.DisplayName
	}
	if issue.Fields.Parent != nil {
		flat["parent"] = issue.Fields.Parent.Key
	}
	if issue.Fields.Resolution != nil {
		flat["resolution"] = issue.Fields.Resolution.Name
	}
	if issue.Fields.DueDate != "" {
		flat["duedate"] = issue.Fields.DueDate
	}
	if issue.Fields.Description != nil {
		flat["description"] = issue.Fields.Description
	}

	// Fix versions
	if len(issue.Fields.FixVersions) > 0 {
		names := make([]string, len(issue.Fields.FixVersions))
		for i, v := range issue.Fields.FixVersions {
			names[i] = v.Name
		}
		flat["fixVersions"] = names
	}

	// Affected versions
	if len(issue.Fields.Versions) > 0 {
		names := make([]string, len(issue.Fields.Versions))
		for i, v := range issue.Fields.Versions {
			names[i] = v.Name
		}
		flat["affectedVersions"] = names
	}

	// Components
	if len(issue.Fields.Components) > 0 {
		names := make([]string, len(issue.Fields.Components))
		for i, c := range issue.Fields.Components {
			names[i] = c.Name
		}
		flat["components"] = names
	}

	// Issue links
	if len(issue.Fields.IssueLinks) > 0 {
		var links []map[string]any
		for _, l := range issue.Fields.IssueLinks {
			link := map[string]any{"id": l.ID, "type": l.Type.Name}
			if l.InwardIssue != nil {
				link["direction"] = "inward"
				link["description"] = l.Type.Inward
				link["key"] = l.InwardIssue.Key
			}
			if l.OutwardIssue != nil {
				link["direction"] = "outward"
				link["description"] = l.Type.Outward
				link["key"] = l.OutwardIssue.Key
			}
			links = append(links, link)
		}
		flat["links"] = links
	}

	// Subtasks
	if len(issue.Fields.Subtasks) > 0 {
		var subs []map[string]any
		for _, s := range issue.Fields.Subtasks {
			sub := map[string]any{"key": s.Key, "summary": s.Fields.Summary}
			if s.Fields.Status != nil {
				sub["status"] = s.Fields.Status.Name
			}
			subs = append(subs, sub)
		}
		flat["subtasks"] = subs
	}

	// Time tracking
	if issue.Fields.TimeTracking != nil {
		tt := map[string]any{}
		if issue.Fields.TimeTracking.OriginalEstimate != "" {
			tt["originalEstimate"] = issue.Fields.TimeTracking.OriginalEstimate
		}
		if issue.Fields.TimeTracking.RemainingEstimate != "" {
			tt["remainingEstimate"] = issue.Fields.TimeTracking.RemainingEstimate
		}
		if issue.Fields.TimeTracking.TimeSpent != "" {
			tt["timeSpent"] = issue.Fields.TimeTracking.TimeSpent
		}
		if len(tt) > 0 {
			flat["timeTracking"] = tt
		}
	}

	// Votes & watches
	if issue.Fields.Votes != nil {
		flat["votes"] = issue.Fields.Votes.Votes
	}
	if issue.Fields.Watches != nil {
		flat["watches"] = issue.Fields.Watches.WatchCount
	}

	// Sprint (extract active sprint name from array)
	if issue.Fields.Sprint != nil {
		flat["sprint"] = extractSprintName(issue.Fields.Sprint)
	}

	// Story points
	if issue.Fields.StoryPts != nil {
		flat["storyPoints"] = *issue.Fields.StoryPts
	}

	// Epic link
	if issue.Fields.EpicLink != "" {
		flat["epicLink"] = issue.Fields.EpicLink
	}

	// Flagged
	if issue.Fields.Flagged != nil {
		flat["flagged"] = isFlagged(issue.Fields.Flagged)
	}

	return flat
}

// extractSprintName gets the active sprint name from the Sprint custom field.
// The field is an array of objects with name, state, etc.
func extractSprintName(data json.RawMessage) string {
	var sprints []struct {
		Name  string `json:"name"`
		State string `json:"state"`
	}
	if json.Unmarshal(data, &sprints) != nil || len(sprints) == 0 {
		return ""
	}
	// Prefer active sprint, fallback to last in array
	for _, s := range sprints {
		if s.State == "active" {
			return s.Name
		}
	}
	return sprints[len(sprints)-1].Name
}

// isFlagged checks if the Flagged custom field contains any values.
func isFlagged(data json.RawMessage) bool {
	var flags []any
	if json.Unmarshal(data, &flags) == nil && len(flags) > 0 {
		return true
	}
	return false
}

// flattenIssues converts a list of Jira issue responses into flat maps.
func flattenIssues(data json.RawMessage) json.RawMessage {
	var searchResp struct {
		Issues        []json.RawMessage `json:"issues"`
		Total         int               `json:"total"`
		NextPageToken string            `json:"nextPageToken"`
		IsLast        bool              `json:"isLast"`
	}

	if json.Unmarshal(data, &searchResp) == nil && searchResp.Issues != nil {
		count := len(searchResp.Issues)
		if searchResp.Total > 0 {
			fmt.Fprintf(os.Stderr, "issues: %d total results\n", searchResp.Total)
		} else if !searchResp.IsLast {
			fmt.Fprintf(os.Stderr, "issues: %d returned (more available)\n", count)
		} else {
			fmt.Fprintf(os.Stderr, "issues: %d results\n", count)
		}

		var flattened []map[string]any
		for _, raw := range searchResp.Issues {
			if flat := flattenIssue(raw); flat != nil {
				flattened = append(flattened, flat)
			}
		}
		if flattened == nil {
			flattened = []map[string]any{}
		}
		out, _ := json.Marshal(flattened)
		return out
	}

	var issues []json.RawMessage
	if json.Unmarshal(data, &issues) == nil {
		var flattened []map[string]any
		for _, raw := range issues {
			if flat := flattenIssue(raw); flat != nil {
				flattened = append(flattened, flat)
			}
		}
		if flattened == nil {
			flattened = []map[string]any{}
		}
		out, _ := json.Marshal(flattened)
		return out
	}

	return data
}

// truncateArray limits a JSON array to n items.
func truncateArray(data json.RawMessage, n int) json.RawMessage {
	var items []json.RawMessage
	if json.Unmarshal(data, &items) != nil || len(items) <= n {
		return data
	}
	out, _ := json.Marshal(items[:n])
	return out
}

// buildBrowseURL constructs a Jira issue URL.
func buildBrowseURL(server, key string) string {
	return strings.TrimRight(server, "/") + "/browse/" + key
}
