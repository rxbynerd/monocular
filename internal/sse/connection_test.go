package sse

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConnectReceivesEvents(t *testing.T) {
	events := []string{
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
		`data: {"directory":"/d","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`,
		"",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}
		for _, line := range events {
			fmt.Fprintln(w, line)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh := make(chan GlobalEvent, 10)
	stateCh := make(chan ConnectionState, 10)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		MaxRetries:   1,
		StaleTimeout: 30 * time.Second,
	}

	go Connect(ctx, cfg, eventCh, stateCh)

	// Should receive Connecting state
	state := <-stateCh
	if state != Connecting {
		t.Errorf("expected Connecting, got %v", state)
	}

	// Should receive Connected after server.connected
	state = <-stateCh
	if state != Connected {
		t.Errorf("expected Connected, got %v", state)
	}

	// Should receive server.connected event
	ev := <-eventCh
	if ev.Payload.Type != "server.connected" {
		t.Errorf("expected server.connected, got %s", ev.Payload.Type)
	}

	// Should receive session.created event
	ev = <-eventCh
	if ev.Payload.Type != "session.created" {
		t.Errorf("expected session.created, got %s", ev.Payload.Type)
	}
}

func TestConnectContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprintln(w, `data: {"payload":{"type":"server.connected","properties":{}}}`)
		fmt.Fprintln(w, "")
		flusher.Flush()
		// Keep connection open until context cancelled
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	eventCh := make(chan GlobalEvent, 10)
	stateCh := make(chan ConnectionState, 10)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		MaxRetries:   50,
		StaleTimeout: 30 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var connectErr error
	go func() {
		defer wg.Done()
		connectErr = Connect(ctx, cfg, eventCh, stateCh)
	}()

	// Wait for connected state
	for state := range stateCh {
		if state == Connected {
			break
		}
	}

	cancel()
	wg.Wait()

	if connectErr != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", connectErr)
	}
}

func TestConnectNon200Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh := make(chan GlobalEvent, 10)
	stateCh := make(chan ConnectionState, 10)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		MaxRetries:   1,
		StaleTimeout: 30 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Connect(ctx, cfg, eventCh, stateCh)
	}()

	// Should see Connecting -> ConnectFailed
	state := <-stateCh
	if state != Connecting {
		t.Errorf("expected Connecting, got %v", state)
	}

	state = <-stateCh
	if state != ConnectFailed {
		t.Errorf("expected ConnectFailed, got %v", state)
	}

	cancel()
	wg.Wait()
}

func TestConnectReconnectOnStreamClose(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		count := callCount
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprintln(w, `data: {"payload":{"type":"server.connected","properties":{}}}`)
		fmt.Fprintln(w, "")
		flusher.Flush()

		if count == 1 {
			// Close immediately on first connect to trigger reconnect
			return
		}
		// Second connect: stay open until context done
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventCh := make(chan GlobalEvent, 20)
	stateCh := make(chan ConnectionState, 20)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		MaxRetries:   5,
		StaleTimeout: 30 * time.Second,
	}

	go Connect(ctx, cfg, eventCh, stateCh)

	// Drain events looking for _reconnected
	foundReconnected := false
	timeout := time.After(8 * time.Second)
	for {
		select {
		case ev := <-eventCh:
			if ev.Payload.Type == "_reconnected" {
				foundReconnected = true
				cancel()
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	if !foundReconnected {
		t.Error("expected _reconnected synthetic event after reconnect")
	}
}

func TestConnectDirectoryFilter(t *testing.T) {
	events := []string{
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
		`data: {"directory":"/proj-a","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`,
		"",
		`data: {"directory":"/proj-b","payload":{"type":"session.created","properties":{"sessionID":"s2"}}}`,
		"",
		`data: {"directory":"/proj-a","payload":{"type":"session.status","properties":{"sessionID":"s1"}}}`,
		"",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no directory query param (client-side filter only)
		if r.URL.Query().Get("directory") != "" {
			t.Error("directory should not be sent as query parameter")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for _, line := range events {
			fmt.Fprintln(w, line)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh := make(chan GlobalEvent, 10)
	stateCh := make(chan ConnectionState, 10)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		Directory:    "/proj-a",
		MaxRetries:   1,
		StaleTimeout: 30 * time.Second,
	}

	go Connect(ctx, cfg, eventCh, stateCh)

	var received []string
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-eventCh:
			received = append(received, ev.Payload.Type)
			if len(received) >= 3 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	// Should have: server.connected (no dir), session.created (proj-a), session.status (proj-a)
	// Should NOT have: session.created for proj-b
	for _, r := range received {
		if strings.Contains(r, "s2") {
			t.Errorf("directory filter should have excluded proj-b event")
		}
	}
	if len(received) < 3 {
		t.Errorf("expected 3 events, got %d: %v", len(received), received)
	}
}

func TestCalcBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		minMs   int
		maxMs   int
	}{
		{1, 750, 1500},     // 1s +-25%
		{2, 1500, 3000},    // 2s +-25%
		{3, 3000, 6000},    // 4s +-25%
		{4, 6000, 12000},   // 8s +-25%
		{5, 12000, 24000},  // 16s +-25%
		{6, 22500, 37500},  // 30s cap +-25%
		{10, 22500, 37500}, // still capped at 30s
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			for i := 0; i < 100; i++ {
				d := calcBackoff(tt.attempt)
				ms := int(d.Milliseconds())
				if ms < tt.minMs || ms > tt.maxMs {
					t.Errorf("calcBackoff(%d) = %dms, want [%d, %d]ms", tt.attempt, ms, tt.minMs, tt.maxMs)
					break
				}
			}
		})
	}
}

func TestConnectStaleDetection(t *testing.T) {
	// Server sends connected event then goes silent to trigger stale detection.
	// Track how many connections the server receives.
	var connCount int
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connCount++
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprintln(w, `data: {"payload":{"type":"server.connected","properties":{}}}`)
		fmt.Fprintln(w, "")
		flusher.Flush()
		// Go silent -- stale detection should close the connection
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eventCh := make(chan GlobalEvent, 10)
	stateCh := make(chan ConnectionState, 20)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		MaxRetries:   3,
		StaleTimeout: 1 * time.Second, // Very short for test
	}

	go Connect(ctx, cfg, eventCh, stateCh)

	// Wait until the server sees at least 2 connections (initial + reconnect after stale)
	timeout := time.After(12 * time.Second)
	for {
		mu.Lock()
		count := connCount
		mu.Unlock()
		if count >= 2 {
			cancel()
			break
		}
		select {
		case <-timeout:
			cancel()
			t.Fatalf("stale detection did not trigger reconnect; server saw %d connections", connCount)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestConnectMaxRetriesExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	eventCh := make(chan GlobalEvent, 10)
	stateCh := make(chan ConnectionState, 20)

	cfg := ConnectionConfig{
		URL:          srv.URL,
		MaxRetries:   2,
		StaleTimeout: 30 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var connectErr error
	go func() {
		defer wg.Done()
		connectErr = Connect(ctx, cfg, eventCh, stateCh)
	}()

	wg.Wait()

	if connectErr == nil {
		t.Fatal("expected error when max retries exhausted")
	}
	if !strings.Contains(connectErr.Error(), "exceeded max retries") {
		t.Errorf("error = %v, want 'exceeded max retries'", connectErr)
	}
}

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{Disconnected, "DISCONNECTED"},
		{Connecting, "CONNECTING"},
		{Connected, "CONNECTED"},
		{ConnectFailed, "CONNECT_FAILED"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnectionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
