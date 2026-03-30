package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

// FormatFunc transforms a value for table display.
type FormatFunc func(any) string

// ColumnDef defines a table column.
type ColumnDef struct {
	Header string
	Key    string
	Format FormatFunc
}

var commandColumns = map[string][]ColumnDef{
	"issues.list": {
		{Header: "KEY", Key: "key"},
		{Header: "TYPE", Key: "type"},
		{Header: "STATUS", Key: "status"},
		{Header: "PRIORITY", Key: "priority"},
		{Header: "ASSIGNEE", Key: "assignee"},
		{Header: "SUMMARY", Key: "summary", Format: truncate60},
	},
	"issues.get": {
		{Header: "KEY", Key: "key"},
		{Header: "TYPE", Key: "type"},
		{Header: "STATUS", Key: "status"},
		{Header: "PRIORITY", Key: "priority"},
		{Header: "ASSIGNEE", Key: "assignee"},
		{Header: "SUMMARY", Key: "summary"},
	},
	"search": {
		{Header: "KEY", Key: "key"},
		{Header: "TYPE", Key: "type"},
		{Header: "STATUS", Key: "status"},
		{Header: "PRIORITY", Key: "priority"},
		{Header: "ASSIGNEE", Key: "assignee"},
		{Header: "SUMMARY", Key: "summary", Format: truncate60},
	},
	"comments.list": {
		{Header: "ID", Key: "id"},
		{Header: "AUTHOR", Key: "author"},
		{Header: "CREATED", Key: "created"},
		{Header: "BODY", Key: "body", Format: truncate60},
	},
	"transitions.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "TO STATUS", Key: "to_status"},
	},
	"sprints.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "STATE", Key: "state"},
		{Header: "START", Key: "startDate"},
		{Header: "END", Key: "endDate"},
	},
	"boards.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
	},
	"epics.list": {
		{Header: "KEY", Key: "key"},
		{Header: "STATUS", Key: "status"},
		{Header: "SUMMARY", Key: "summary", Format: truncate60},
	},
	"projects.list": {
		{Header: "KEY", Key: "key"},
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "projectTypeKey"},
	},
	"users.list": {
		{Header: "ACCOUNT ID", Key: "accountId"},
		{Header: "NAME", Key: "displayName"},
		{Header: "EMAIL", Key: "emailAddress"},
		{Header: "ACTIVE", Key: "active"},
	},
	"fields.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "CUSTOM", Key: "custom"},
		{Header: "TYPE", Key: "type"},
	},
	"labels.list": {
		{Header: "LABEL", Key: "label"},
	},
	"config.list": {
		{Header: "NAME", Key: "name"},
		{Header: "EMAIL", Key: "email"},
		{Header: "TOKEN", Key: "token"},
		{Header: "SERVER", Key: "server"},
		{Header: "DEFAULT", Key: "default"},
	},
	"versions.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "STATUS", Key: "status"},
		{Header: "RELEASE DATE", Key: "releaseDate"},
	},
	"components.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "LEAD", Key: "lead"},
	},
	"statuses.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "CATEGORY", Key: "category"},
	},
	"filters.list": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "OWNER", Key: "owner"},
		{Header: "JQL", Key: "jql", Format: truncate60},
	},
	"remote-links.list": {
		{Header: "ID", Key: "id"},
		{Header: "TITLE", Key: "title"},
		{Header: "URL", Key: "url", Format: truncate60},
	},
	"permissions.mine": {
		{Header: "PERMISSION", Key: "permission"},
		{Header: "HAVE", Key: "have"},
	},
	"votes.list": {
		{Header: "ACCOUNT ID", Key: "accountId"},
		{Header: "NAME", Key: "displayName"},
	},
	"properties.list": {
		{Header: "KEY", Key: "key"},
	},
}

func printTable(command string, data json.RawMessage) error {
	columns, ok := commandColumns[command]
	if !ok {
		return fmt.Errorf("no table definition for %s", command)
	}

	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		var single map[string]any
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return fmt.Errorf("cannot render as table")
		}
		rows = []map[string]any{single}
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	if term.IsTerminal(int(os.Stdout.Fd())) {
		t.SetStyle(table.StyleLight)
	} else {
		t.SetStyle(table.StyleDefault)
	}

	header := make(table.Row, len(columns))
	for i, col := range columns {
		header[i] = col.Header
	}
	t.AppendHeader(header)

	for _, row := range rows {
		r := make(table.Row, len(columns))
		for i, col := range columns {
			if col.Format != nil {
				r[i] = col.Format(row[col.Key])
			} else {
				r[i] = formatValue(row[col.Key])
			}
		}
		t.AppendRow(r)
	}

	t.Render()
	return nil
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ", ")
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func truncate60(v any) string {
	s := formatValue(v)
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}
