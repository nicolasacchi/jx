package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client is an HTTP client for the Jira REST API v3.
type Client struct {
	http    *http.Client
	email   string
	token   string
	server  string // e.g. https://1000farmacie.atlassian.net
	verbose bool
}

// New creates a Jira API client.
func New(email, token, server string, verbose bool) *Client {
	server = strings.TrimRight(server, "/")
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		email:   email,
		token:   token,
		server:  server,
		verbose: verbose,
	}
}

// Server returns the configured server URL.
func (c *Client) Server() string {
	return c.server
}

func (c *Client) authHeader() string {
	encoded := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.token))
	return "Basic " + encoded
}

func (c *Client) do(ctx context.Context, method, path string, body any, params url.Values) (json.RawMessage, error) {
	u := c.server + "/" + strings.TrimLeft(path, "/")
	if params != nil && len(params) > 0 {
		u += "?" + params.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "→ %s %s\n", method, u)
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "← %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, respBody)
	}

	// DELETE with 204 returns no body
	if resp.StatusCode == 204 || len(respBody) == 0 {
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(respBody), nil
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 429 || attempt == maxRetries {
			return resp, nil
		}
		resp.Body.Close()
		wait := time.Duration(1<<uint(attempt)) * time.Second
		if c.verbose {
			fmt.Fprintf(os.Stderr, "rate limited, retrying in %s\n", wait)
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		// Re-create request body for retry if needed
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}
	}
	return nil, fmt.Errorf("max retries exceeded")
}

func parseError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode, RawBody: string(body)}
	var errResp struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if json.Unmarshal(body, &errResp) == nil {
		apiErr.Messages = errResp.ErrorMessages
		apiErr.Errors = errResp.Errors
	}
	return apiErr
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	return c.do(ctx, "GET", path, nil, params)
}

// GetBinary performs a GET and returns the raw response bytes (for binary content
// like attachment downloads). Uses a separate http.Client with a 5-minute timeout to
// handle large files (videos, archives) that would exceed the 30s default. Skips the
// JSON parsing wrapper that the standard `do()` path applies.
func (c *Client) GetBinary(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := c.server + "/" + strings.TrimLeft(path, "/")
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	// Intentionally no Accept header — let the server pick the content-type for binary.

	if c.verbose {
		fmt.Fprintf(os.Stderr, "→ GET %s (binary)\n", u)
	}

	binaryClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := binaryClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read binary response: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "← %d (%d bytes)\n", resp.StatusCode, len(body))
	}

	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, body)
	}
	return body, nil
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.do(ctx, "POST", path, body, nil)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.do(ctx, "PUT", path, body, nil)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.do(ctx, "DELETE", path, nil, nil)
	return err
}

// PostRaw performs a POST with a pre-built raw body (for multipart, etc.).
func (c *Client) PostRaw(ctx context.Context, path string, bodyReader io.Reader, contentType string) (json.RawMessage, error) {
	u := c.server + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, "POST", u, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Atlassian-Token", "no-check") // required for attachments

	if c.verbose {
		fmt.Fprintf(os.Stderr, "→ POST %s (raw: %s)\n", u, contentType)
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "← %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, respBody)
	}
	if len(respBody) == 0 {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage(respBody), nil
}
