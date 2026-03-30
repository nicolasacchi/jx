package jql

import (
	"fmt"
	"strings"
)

// Builder constructs JQL queries from structured inputs.
type Builder struct {
	clauses []string
	orderBy string
}

// New creates a new JQL builder.
func New() *Builder {
	return &Builder{}
}

// Project adds a project clause.
func (b *Builder) Project(key string) *Builder {
	if key != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("project = %s", key))
	}
	return b
}

// Status adds a status clause.
func (b *Builder) Status(status string) *Builder {
	if status != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("status = %q", status))
	}
	return b
}

// StatusIn adds a status IN clause for multiple statuses.
func (b *Builder) StatusIn(statuses []string) *Builder {
	if len(statuses) > 0 {
		quoted := make([]string, len(statuses))
		for i, s := range statuses {
			quoted[i] = fmt.Sprintf("%q", s)
		}
		b.clauses = append(b.clauses, fmt.Sprintf("status in (%s)", strings.Join(quoted, ", ")))
	}
	return b
}

// Type adds an issue type clause.
func (b *Builder) Type(issueType string) *Builder {
	if issueType != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("issuetype = %q", issueType))
	}
	return b
}

// Assignee adds an assignee clause.
func (b *Builder) Assignee(user string) *Builder {
	if user != "" {
		if strings.ToLower(user) == "currentuser()" || strings.ToLower(user) == "me" {
			b.clauses = append(b.clauses, "assignee = currentUser()")
		} else if strings.ToLower(user) == "unassigned" || user == "none" {
			b.clauses = append(b.clauses, "assignee is EMPTY")
		} else {
			b.clauses = append(b.clauses, fmt.Sprintf("assignee = %q", user))
		}
	}
	return b
}

// UpdatedSince adds an updated >= clause. Accepts Jira relative dates like "-7d".
func (b *Builder) UpdatedSince(since string) *Builder {
	if since != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("updated >= %q", since))
	}
	return b
}

// CreatedSince adds a created >= clause.
func (b *Builder) CreatedSince(since string) *Builder {
	if since != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("created >= %q", since))
	}
	return b
}

// Labels adds a labels clause (AND — all labels must match).
func (b *Builder) Labels(labels []string) *Builder {
	for _, l := range labels {
		if l != "" {
			b.clauses = append(b.clauses, fmt.Sprintf("labels = %q", l))
		}
	}
	return b
}

// Priority adds a priority clause.
func (b *Builder) Priority(priority string) *Builder {
	if priority != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("priority = %q", priority))
	}
	return b
}

// Parent adds a parent clause (for subtasks).
func (b *Builder) Parent(parentKey string) *Builder {
	if parentKey != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("parent = %s", parentKey))
	}
	return b
}

// Epic adds an epic link clause.
func (b *Builder) Epic(epicKey string) *Builder {
	if epicKey != "" {
		b.clauses = append(b.clauses, fmt.Sprintf(`"Epic Link" = %s`, epicKey))
	}
	return b
}

// Sprint adds a sprint clause. Supports "current", "open", or a sprint name.
func (b *Builder) Sprint(sprint string) *Builder {
	if sprint != "" {
		switch strings.ToLower(sprint) {
		case "current", "active", "open":
			b.clauses = append(b.clauses, "sprint in openSprints()")
		case "closed":
			b.clauses = append(b.clauses, "sprint in closedSprints()")
		case "future":
			b.clauses = append(b.clauses, "sprint in futureSprints()")
		default:
			b.clauses = append(b.clauses, fmt.Sprintf("sprint = %q", sprint))
		}
	}
	return b
}

// Resolution adds a resolution clause.
func (b *Builder) Resolution(resolution string) *Builder {
	if resolution != "" {
		if strings.ToLower(resolution) == "unresolved" || resolution == "none" {
			b.clauses = append(b.clauses, "resolution is EMPTY")
		} else {
			b.clauses = append(b.clauses, fmt.Sprintf("resolution = %q", resolution))
		}
	}
	return b
}

// Text adds a text search clause.
func (b *Builder) Text(query string) *Builder {
	if query != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("text ~ %q", query))
	}
	return b
}

// Component adds a component clause.
func (b *Builder) Component(name string) *Builder {
	if name != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("component = %q", name))
	}
	return b
}

// FixVersion adds a fix version clause.
func (b *Builder) FixVersion(version string) *Builder {
	if version != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("fixVersion = %q", version))
	}
	return b
}

// DueBefore adds a duedate <= clause.
func (b *Builder) DueBefore(date string) *Builder {
	if date != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("duedate <= %q", date))
	}
	return b
}

// DueAfter adds a duedate >= clause.
func (b *Builder) DueAfter(date string) *Builder {
	if date != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("duedate >= %q", date))
	}
	return b
}

// Raw adds a raw JQL clause as-is.
func (b *Builder) Raw(clause string) *Builder {
	if clause != "" {
		b.clauses = append(b.clauses, clause)
	}
	return b
}

// OrderBy sets the ORDER BY clause.
func (b *Builder) OrderBy(field, direction string) *Builder {
	if field != "" {
		if direction == "" {
			direction = "DESC"
		}
		b.orderBy = fmt.Sprintf("ORDER BY %s %s", field, strings.ToUpper(direction))
	}
	return b
}

// Build constructs the final JQL string.
func (b *Builder) Build() string {
	jql := strings.Join(b.clauses, " AND ")
	if b.orderBy != "" {
		if jql != "" {
			jql += " " + b.orderBy
		} else {
			jql = b.orderBy
		}
	}
	return jql
}

// IsEmpty returns true if no clauses have been added.
func (b *Builder) IsEmpty() bool {
	return len(b.clauses) == 0 && b.orderBy == ""
}
