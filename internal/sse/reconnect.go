package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SessionInfo holds session data fetched via REST on reconnect.
type SessionInfo struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Directory string `json:"directory"`
}

// FetchSessions fetches the current session list from the OpenCode REST API.
// Used after SSE reconnection to recover accurate session state.
// baseURL should be the server root (e.g. "http://127.0.0.1:4096").
func FetchSessions(ctx context.Context, baseURL string) ([]SessionInfo, error) {
	url := baseURL + "/session"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var sessions []SessionInfo
	if err := json.Unmarshal(body, &sessions); err != nil {
		return nil, fmt.Errorf("parsing sessions: %w", err)
	}

	return sessions, nil
}
