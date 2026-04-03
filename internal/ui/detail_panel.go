package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"github.com/rxbynerd/monocular/internal/model"
	"github.com/rxbynerd/monocular/internal/sse"
)

// DetailPanel renders the right panel with selected event detail.
type DetailPanel struct {
	viewport viewport.Model
	width    int
	height   int
}

func NewDetailPanel() DetailPanel {
	vp := viewport.New()
	return DetailPanel{
		viewport: vp,
	}
}

func (d *DetailPanel) SetSize(w, h int) {
	d.width = w
	d.height = h
	d.viewport.SetWidth(w)
	d.viewport.SetHeight(h)
}

func (d *DetailPanel) ScrollDown(n int) {
	d.viewport.ScrollDown(n)
}

func (d *DetailPanel) ScrollUp(n int) {
	d.viewport.ScrollUp(n)
}

func (d *DetailPanel) GotoTop() {
	d.viewport.GotoTop()
}

func (d *DetailPanel) GotoBottom() {
	d.viewport.GotoBottom()
}

func (d DetailPanel) View(state *model.DashboardState) string {
	events := state.Events
	idx := state.UI.SelectedEventIdx

	if len(events) == 0 || idx < 0 || idx >= len(events) {
		return styleDim.Render("  Select an event to view details")
	}

	ev := events[idx]

	var lines []string

	// Header
	lines = append(lines, styleTitle.Render(ev.Type))
	lines = append(lines, styleDim.Render(strings.Repeat("-", d.width-2)))

	// Category
	badge := CategoryStyle(ev.Category).Render(sse.CategoryBadge(ev.Category))
	lines = append(lines, fmt.Sprintf("Category: %s", badge))

	if ev.Directory != "" {
		lines = append(lines, fmt.Sprintf("Directory: %s", ev.Directory))
	}

	lines = append(lines, fmt.Sprintf("Time: %s", ev.Timestamp.Format("15:04:05.000")))
	lines = append(lines, "")

	// Formatted properties
	if ev.Properties != nil {
		lines = append(lines, styleTitle.Render("Properties:"))
		lines = append(lines, renderProperties(ev.Properties, 1)...)
		lines = append(lines, "")
	}

	// Raw JSON
	lines = append(lines, styleTitle.Render("Raw JSON:"))
	raw, err := json.MarshalIndent(ev.Properties, "", "  ")
	if err == nil {
		lines = append(lines, styleDim.Render(string(raw)))
	}

	content := strings.Join(lines, "\n")
	d.viewport.SetContent(content)
	return d.viewport.View()
}

func renderProperties(m map[string]any, indent int) []string {
	var lines []string
	prefix := strings.Repeat("  ", indent)

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case map[string]any:
			lines = append(lines, fmt.Sprintf("%s%s:", prefix, k))
			lines = append(lines, renderProperties(val, indent+1)...)
		case []any:
			lines = append(lines, fmt.Sprintf("%s%s: [%d items]", prefix, k, len(val)))
		case string:
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, k, val))
		case float64:
			if val == float64(int(val)) {
				lines = append(lines, fmt.Sprintf("%s%s: %d", prefix, k, int(val)))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s: %g", prefix, k, val))
			}
		case bool:
			lines = append(lines, fmt.Sprintf("%s%s: %v", prefix, k, val))
		default:
			lines = append(lines, fmt.Sprintf("%s%s: %v", prefix, k, v))
		}
	}
	return lines
}
