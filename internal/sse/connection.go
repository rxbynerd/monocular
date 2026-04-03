package sse

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// ConnectionState represents the current state of the SSE connection.
type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connecting
	Connected
	ConnectFailed
)

func (s ConnectionState) String() string {
	switch s {
	case Disconnected:
		return "DISCONNECTED"
	case Connecting:
		return "CONNECTING"
	case Connected:
		return "CONNECTED"
	case ConnectFailed:
		return "CONNECT_FAILED"
	default:
		return "UNKNOWN"
	}
}

// ConnectionConfig configures the SSE connection.
type ConnectionConfig struct {
	URL          string        // Full URL to /global/event endpoint
	Directory    string        // Optional client-side directory filter
	MaxRetries   int           // Default: 50
	StaleTimeout time.Duration // Default: 30s -- force reconnect if no event received
}

// DefaultConfig returns a ConnectionConfig with sensible defaults.
func DefaultConfig(baseURL string) ConnectionConfig {
	return ConnectionConfig{
		URL:          baseURL + "/global/event",
		MaxRetries:   50,
		StaleTimeout: 30 * time.Second,
	}
}

// Connect runs the SSE connection loop. It blocks until ctx is cancelled.
// Events are sent to eventCh. State changes are sent to stateCh.
// On stream break, it reconnects with exponential backoff.
func Connect(ctx context.Context, cfg ConnectionConfig, eventCh chan<- GlobalEvent, stateCh chan<- ConnectionState) error {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 50
	}
	if cfg.StaleTimeout == 0 {
		cfg.StaleTimeout = 30 * time.Second
	}

	retries := 0
	isReconnect := false

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sendState(ctx, stateCh, Connecting)

		err := streamEvents(ctx, cfg, eventCh, stateCh, isReconnect)
		if ctx.Err() != nil {
			sendState(ctx, stateCh, Disconnected)
			return ctx.Err()
		}

		isReconnect = true
		retries++
		if retries > cfg.MaxRetries {
			sendState(ctx, stateCh, Disconnected)
			return fmt.Errorf("exceeded max retries (%d): last error: %w", cfg.MaxRetries, err)
		}

		sendState(ctx, stateCh, ConnectFailed)

		backoff := calcBackoff(retries)
		select {
		case <-ctx.Done():
			sendState(ctx, stateCh, Disconnected)
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}

func streamEvents(ctx context.Context, cfg ConnectionConfig, eventCh chan<- GlobalEvent, stateCh chan<- ConnectionState, isReconnect bool) error {
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 0} // no timeout for SSE
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Track last event time for stale detection
	var mu sync.Mutex
	lastEventTime := time.Now()

	// Stale detection goroutine
	staleCtx, staleCancel := context.WithCancel(ctx)
	defer staleCancel()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-staleCtx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				elapsed := time.Since(lastEventTime)
				mu.Unlock()
				if elapsed > cfg.StaleTimeout {
					resp.Body.Close()
					return
				}
			}
		}
	}()

	parser := NewSSEParser()
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line := scanner.Text()
		event, err := parser.FeedLine(line)
		if err != nil {
			continue // skip malformed events
		}
		if event == nil {
			continue
		}

		mu.Lock()
		lastEventTime = time.Now()
		mu.Unlock()

		// Handle server.connected: transition to Connected state
		if event.Payload.Type == "server.connected" {
			sendState(ctx, stateCh, Connected)

			if isReconnect {
				// Send synthetic reconnected event
				sendEvent(ctx, eventCh, GlobalEvent{
					Payload: EventPayload{
						Type:       "_reconnected",
						Properties: map[string]any{},
					},
				})
			}
		}

		// Apply client-side directory filter
		if cfg.Directory != "" && event.Directory != "" && event.Directory != cfg.Directory {
			continue
		}

		sendEvent(ctx, eventCh, *event)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}
	return fmt.Errorf("stream ended")
}

func sendState(ctx context.Context, ch chan<- ConnectionState, state ConnectionState) {
	select {
	case ch <- state:
	case <-ctx.Done():
	}
}

func sendEvent(ctx context.Context, ch chan<- GlobalEvent, event GlobalEvent) {
	select {
	case ch <- event:
	case <-ctx.Done():
	}
}

// calcBackoff returns a duration with exponential backoff and jitter.
// Base: 1s, multiplied by 2^(attempt-1), capped at 30s.
func calcBackoff(attempt int) time.Duration {
	base := time.Second
	max := 30 * time.Second

	backoff := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if backoff > max {
		backoff = max
	}

	// Add jitter: +-25%
	jitter := time.Duration(float64(backoff) * (0.75 + rand.Float64()*0.5))
	return jitter
}
