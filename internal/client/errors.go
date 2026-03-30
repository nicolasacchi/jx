package client

import "fmt"

// APIError represents an error from the Jira REST API.
type APIError struct {
	StatusCode int
	Messages   []string
	Errors     map[string]string
	RawBody    string
}

func (e *APIError) Error() string {
	if len(e.Messages) > 0 {
		return fmt.Sprintf("jira: %d — %s", e.StatusCode, e.Messages[0])
	}
	for field, msg := range e.Errors {
		return fmt.Sprintf("jira: %d — %s: %s", e.StatusCode, field, msg)
	}
	if e.RawBody != "" {
		body := e.RawBody
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return fmt.Sprintf("jira: %d — %s", e.StatusCode, body)
	}
	return fmt.Sprintf("jira: %d", e.StatusCode)
}

// ExitCode returns the appropriate process exit code.
func (e *APIError) ExitCode() int {
	switch {
	case e.StatusCode == 401 || e.StatusCode == 403:
		return 3 // auth error
	case e.StatusCode == 404:
		return 4 // not found
	default:
		return 1 // general API error
	}
}
