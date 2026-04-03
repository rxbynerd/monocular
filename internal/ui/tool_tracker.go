package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/format"
	"github.com/rxbynerd/monocular/internal/model"
)

// ToolTracker renders the active tool executions below the session panel.
type ToolTracker struct {
	width int
}

func NewToolTracker() ToolTracker {
	return ToolTracker{}
}

func (t *ToolTracker) SetWidth(w int) {
	t.width = w
}

func (t ToolTracker) View(state *model.DashboardState, now time.Time) string {
	var tools []model.ToolExecution
	for _, id := range state.SessionOrder {
		entry, ok := state.Sessions[id]
		if !ok {
			continue
		}
		tools = append(tools, entry.ActiveTools...)
	}

	if len(tools) == 0 {
		return ""
	}

	header := styleDim.Render("  -- Active Tools --")
	var lines []string
	lines = append(lines, header)

	contentWidth := t.width - 4
	for _, tool := range tools {
		elapsed := format.Duration(now.Sub(tool.StartedAt))
		sessionShort := format.ShortID(tool.SessionID, 10)

		var toolStyle lipgloss.Style
		switch tool.Status {
		case "pending":
			toolStyle = styleToolPending
		case "running":
			toolStyle = styleToolRunning
		default:
			toolStyle = styleToolPending
		}

		line := fmt.Sprintf("  %s (%s) %s",
			toolStyle.Render(format.Truncate(tool.Tool, 15)),
			styleDim.Render(sessionShort),
			styleDim.Render(elapsed),
		)
		lines = append(lines, format.Truncate(line, contentWidth))
	}

	return strings.Join(lines, "\n")
}
