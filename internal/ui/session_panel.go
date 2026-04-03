package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/format"
	"github.com/rxbynerd/monocular/internal/model"
)

// SessionPanel renders the left panel with session list and status badges.
type SessionPanel struct {
	viewport    viewport.Model
	width       int
	height      int
	selectedIdx int
}

func NewSessionPanel() SessionPanel {
	vp := viewport.New()
	return SessionPanel{
		viewport: vp,
	}
}

func (p *SessionPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.SetWidth(w)
	p.viewport.SetHeight(h)
}

func (p *SessionPanel) ScrollDown(n int) {
	p.selectedIdx += n
}

func (p *SessionPanel) ScrollUp(n int) {
	p.selectedIdx -= n
	if p.selectedIdx < 0 {
		p.selectedIdx = 0
	}
}

func (p *SessionPanel) GotoTop() {
	p.selectedIdx = 0
}

func (p *SessionPanel) GotoBottom(count int) {
	p.selectedIdx = count - 1
	if p.selectedIdx < 0 {
		p.selectedIdx = 0
	}
}

func (p *SessionPanel) SelectedIdx() int {
	return p.selectedIdx
}

func (p SessionPanel) View(state *model.DashboardState) string {
	if len(state.SessionOrder) == 0 {
		return styleDim.Render("  No sessions")
	}

	// Clamp selection
	if p.selectedIdx >= len(state.SessionOrder) {
		p.selectedIdx = len(state.SessionOrder) - 1
	}

	var lines []string
	contentWidth := p.width - 2

	for i, id := range state.SessionOrder {
		entry, ok := state.Sessions[id]
		if !ok {
			continue
		}

		badge := SessionStatusBadge(entry.Status.Type)
		title := format.Truncate(entry.Title, contentWidth-12)
		if title == "" {
			title = format.ShortID(entry.ID, contentWidth-12)
		}

		// Streaming indicator
		streaming := ""
		if state.UI.StreamingIndicator != nil {
			if _, ok := state.UI.StreamingIndicator[id]; ok {
				streaming = styleToolRunning.Render(" ~")
			}
		}

		line := fmt.Sprintf("  %s %s%s", badge, title, streaming)

		selected := i == p.selectedIdx
		if selected {
			prefix := lipgloss.NewStyle().Bold(true).Foreground(colorCyan).Render("> ")
			line = prefix + line[2:] // replace leading spaces with arrow
		}
		if state.UI.SelectedSessionID == id {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Width(contentWidth).
				Render(line)
		}

		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	p.viewport.SetContent(content)
	return p.viewport.View()
}

// SelectedSessionID returns the session ID at the current selection index.
func (p SessionPanel) SelectedSessionID(state *model.DashboardState) string {
	if p.selectedIdx < 0 || p.selectedIdx >= len(state.SessionOrder) {
		return ""
	}
	return state.SessionOrder[p.selectedIdx]
}
