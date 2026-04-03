package model

import (
	"github.com/rxbynerd/monocular/internal/sse"
)

// SSEEventMsg wraps an SSE event for the Bubble Tea message loop.
type SSEEventMsg struct {
	Event sse.GlobalEvent
}

// ConnectionStateMsg carries a connection state change.
type ConnectionStateMsg struct {
	State sse.ConnectionState
}

// TickMsg is fired every 1s for elapsed-time counters.
type TickMsg struct{}

// SessionsRefreshedMsg carries session data fetched via REST on reconnect.
type SessionsRefreshedMsg struct {
	Sessions []sse.SessionInfo
}
