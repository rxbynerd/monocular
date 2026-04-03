package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/format"
	"github.com/rxbynerd/monocular/internal/model"
	"github.com/rxbynerd/monocular/internal/sse"
)

// EventLog renders the scrollable center panel with event entries.
type EventLog struct {
	viewport    viewport.Model
	width       int
	height      int
	followMode  bool
	selectedIdx int
}

func NewEventLog() EventLog {
	vp := viewport.New()
	return EventLog{
		viewport:   vp,
		followMode: true,
	}
}

func (e *EventLog) SetSize(w, h int) {
	e.width = w
	e.height = h
	e.viewport.SetWidth(w)
	e.viewport.SetHeight(h)
}

func (e *EventLog) ScrollDown(n int) {
	e.viewport.ScrollDown(n)
	e.followMode = false
}

func (e *EventLog) ScrollUp(n int) {
	e.viewport.ScrollUp(n)
	e.followMode = false
}

func (e *EventLog) GotoTop() {
	e.viewport.GotoTop()
	e.followMode = false
}

func (e *EventLog) GotoBottom() {
	e.viewport.GotoBottom()
	e.followMode = true
}

func (e *EventLog) SelectedIdx() int {
	return e.selectedIdx
}

func (e *EventLog) SetSelectedIdx(idx int) {
	e.selectedIdx = idx
}

func (e EventLog) View(state *model.DashboardState) string {
	events := filteredEvents(state)

	if len(events) == 0 {
		empty := styleDim.Render("  No events yet...")
		e.viewport.SetContent(empty)
		return e.viewport.View()
	}

	var lines []string
	contentWidth := e.width - 2 // account for padding

	for i, ev := range events {
		line := renderEventLine(ev, contentWidth, i == e.selectedIdx)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	e.viewport.SetContent(content)

	if e.followMode {
		e.viewport.GotoBottom()
	}

	return e.viewport.View()
}

func filteredEvents(state *model.DashboardState) []model.EventLogEntry {
	events := state.Events
	filter := state.UI.Filter
	search := state.UI.SearchQuery
	sessionFilter := state.UI.SelectedSessionID

	if filter == nil && search == "" && sessionFilter == "" {
		return events
	}

	var result []model.EventLogEntry
	for _, ev := range events {
		if filter != nil && !filter[ev.Category] {
			continue
		}
		if sessionFilter != "" {
			sid := getSessionIDFromProps(ev.Properties)
			if sid != "" && sid != sessionFilter {
				continue
			}
		}
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(ev.Summary), searchLower) &&
				!strings.Contains(strings.ToLower(ev.Type), searchLower) {
				continue
			}
		}
		result = append(result, ev)
	}
	return result
}

func getSessionIDFromProps(props map[string]any) string {
	if props == nil {
		return ""
	}
	if sid, ok := props["sessionID"].(string); ok {
		return sid
	}
	return ""
}

func renderEventLine(ev model.EventLogEntry, width int, selected bool) string {
	timestamp := styleDim.Render(format.Timestamp(ev.Timestamp))
	badge := CategoryStyle(ev.Category).Render(sse.CategoryBadge(ev.Category))

	// Calculate remaining width for summary
	// timestamp(8) + space(1) + badge(~10) + space(1)
	summaryWidth := width - 22
	if summaryWidth < 10 {
		summaryWidth = 10
	}
	summary := format.Truncate(ev.Summary, summaryWidth)

	line := fmt.Sprintf("%s %s %s", timestamp, badge, summary)

	if selected {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Width(width).
			Render(line)
	}
	return line
}
