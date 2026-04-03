package jsonmode

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/signal"
	"syscall"

	"github.com/rxbynerd/monocular/internal/sse"
)

// Config holds the configuration for JSON mode output.
type Config struct {
	URL       string
	Directory string
	Filter    map[sse.EventCategory]bool // nil = show all
}

// Run connects to the SSE stream and writes each event as NDJSON to w.
// It blocks until ctx is cancelled or a write error occurs.
func Run(ctx context.Context, cfg Config, w io.Writer) error {
	// Handle SIGPIPE gracefully (broken pipe from head, jq, etc.)
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGPIPE)
	defer stop()

	sseConfig := sse.DefaultConfig(cfg.URL)
	sseConfig.Directory = cfg.Directory

	eventCh := make(chan sse.GlobalEvent, 100)
	stateCh := make(chan sse.ConnectionState, 10)

	errCh := make(chan error, 1)
	go func() {
		errCh <- sse.Connect(ctx, sseConfig, eventCh, stateCh)
	}()

	encoder := json.NewEncoder(w)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case ev := <-eventCh:
			// Skip internal synthetic events
			if ev.Payload.Type == "_reconnected" || ev.Payload.Type == "_sessions_refreshed" {
				continue
			}

			// Apply category filter
			if cfg.Filter != nil {
				cat := sse.Categorize(ev.Payload.Type)
				if !cfg.Filter[cat] {
					continue
				}
			}

			if err := encoder.Encode(ev); err != nil {
				if errors.Is(err, syscall.EPIPE) {
					return nil
				}
				return err
			}
		}
	}
}
