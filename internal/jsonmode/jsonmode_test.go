package jsonmode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rxbynerd/monocular/internal/sse"
)

func makeSSEServer(events []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		for _, line := range events {
			fmt.Fprintln(w, line)
			flusher.Flush()
		}
	}))
}

func TestRunOutputsNDJSON(t *testing.T) {
	srv := makeSSEServer([]string{
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
		`data: {"directory":"/d","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`,
		"",
		`data: {"directory":"/d","payload":{"type":"session.status","properties":{"sessionID":"s1","status":{"type":"busy"}}}}`,
		"",
	})
	defer srv.Close()

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{URL: srv.URL}

	go func() {
		// Wait for events to be written then cancel
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	Run(ctx, cfg, &buf)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 NDJSON lines, got %d: %s", len(lines), buf.String())
	}

	// Each line should be valid JSON
	for i, line := range lines {
		var ev sse.GlobalEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("line %d is not valid JSON: %s", i, line)
		}
	}
}

func TestRunDirectoryFilter(t *testing.T) {
	srv := makeSSEServer([]string{
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
		`data: {"directory":"/proj-a","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`,
		"",
		`data: {"directory":"/proj-b","payload":{"type":"session.created","properties":{"sessionID":"s2"}}}`,
		"",
	})
	defer srv.Close()

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := Config{
		URL:       srv.URL,
		Directory: "/proj-a",
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	Run(ctx, cfg, &buf)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev sse.GlobalEvent
		json.Unmarshal([]byte(line), &ev)
		// Events with directory should only be proj-a; server.connected has no dir
		if ev.Directory != "" && ev.Directory != "/proj-a" {
			t.Errorf("expected only /proj-a events, got directory=%s", ev.Directory)
		}
	}
}

func TestRunCategoryFilter(t *testing.T) {
	srv := makeSSEServer([]string{
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
		`data: {"directory":"/d","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`,
		"",
		`data: {"directory":"/d","payload":{"type":"file.edited","properties":{"file":"test.go"}}}`,
		"",
	})
	defer srv.Close()

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := Config{
		URL: srv.URL,
		Filter: map[sse.EventCategory]bool{
			sse.CategorySession: true,
			// file and infra not included
		},
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	Run(ctx, cfg, &buf)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev sse.GlobalEvent
		json.Unmarshal([]byte(line), &ev)
		cat := sse.Categorize(ev.Payload.Type)
		if cat != sse.CategorySession {
			t.Errorf("expected only session events, got %s (%s)", ev.Payload.Type, cat)
		}
	}
}

func TestRunContextCancellation(t *testing.T) {
	srv := makeSSEServer([]string{
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
	})
	defer srv.Close()

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := Run(ctx, Config{URL: srv.URL}, &buf)
	if err != nil {
		t.Errorf("expected nil error on cancellation, got %v", err)
	}
}
