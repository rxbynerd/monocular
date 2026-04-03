package ui

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/format"
	"github.com/rxbynerd/monocular/internal/model"
	"github.com/rxbynerd/monocular/internal/sse"
)

// ConnectionBar renders the top bar with connection info, uptime, event count, and cost.
type ConnectionBar struct {
	width int
}

func NewConnectionBar() ConnectionBar {
	return ConnectionBar{}
}

func (b *ConnectionBar) SetWidth(w int) {
	b.width = w
}

func (b ConnectionBar) View(state *model.DashboardState, now time.Time) string {
	conn := state.Connection

	// Connection state
	var stateStr string
	switch conn.State {
	case sse.Connected:
		stateStr = styleConnected.Render("CONNECTED")
	case sse.Connecting:
		stateStr = styleConnecting.Render("CONNECTING")
	case sse.ConnectFailed:
		stateStr = styleConnectFailed.Render("CONNECT_FAILED")
	default:
		stateStr = styleDisconnected.Render("DISCONNECTED")
	}

	// Uptime
	uptime := ""
	if conn.State == sse.Connected && !conn.ConnectedAt.IsZero() {
		d := now.Sub(conn.ConnectedAt)
		uptime = format.Duration(d)
	}

	// Build bar
	parts := []string{
		fmt.Sprintf("Connection: %s", conn.URL),
		stateStr,
	}
	if uptime != "" {
		parts = append(parts, uptime)
	}
	parts = append(parts, fmt.Sprintf("%s events", format.Tokens(state.Counters.TotalEvents)))
	if state.Counters.TotalCost > 0 {
		parts = append(parts, format.Cost(state.Counters.TotalCost))
	}
	if conn.ReconnectCount > 0 {
		parts = append(parts, fmt.Sprintf("reconn:%d", conn.ReconnectCount))
	}

	content := ""
	for i, p := range parts {
		if i > 0 {
			content += styleDim.Render(" | ")
		}
		content += p
	}

	style := lipgloss.NewStyle().
		Width(b.width).
		Padding(0, 1).
		Background(lipgloss.Color("235"))

	return style.Render(content)
}
