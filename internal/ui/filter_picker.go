package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/model"
	"github.com/rxbynerd/monocular/internal/sse"
)

// FilterPicker renders an overlay for toggling event category filters.
type FilterPicker struct {
	selectedIdx int
	categories  []sse.EventCategory
}

func NewFilterPicker() FilterPicker {
	return FilterPicker{
		categories: sse.AllCategories(),
	}
}

func (f *FilterPicker) MoveDown() {
	f.selectedIdx++
	if f.selectedIdx >= len(f.categories) {
		f.selectedIdx = 0
	}
}

func (f *FilterPicker) MoveUp() {
	f.selectedIdx--
	if f.selectedIdx < 0 {
		f.selectedIdx = len(f.categories) - 1
	}
}

// Toggle toggles the selected category in the filter map.
// If filter is nil, it initializes it with all categories enabled except the selected one.
func (f *FilterPicker) Toggle(state *model.DashboardState) {
	if state.UI.Filter == nil {
		state.UI.Filter = make(map[sse.EventCategory]bool)
		for _, cat := range f.categories {
			state.UI.Filter[cat] = true
		}
	}
	cat := f.categories[f.selectedIdx]
	state.UI.Filter[cat] = !state.UI.Filter[cat]
}

func (f FilterPicker) View(state *model.DashboardState, width, height int) string {
	var lines []string
	lines = append(lines, styleTitle.Render("  Filter by Category"))
	lines = append(lines, styleDim.Render("  Space: toggle, f/Esc: close"))
	lines = append(lines, "")

	for i, cat := range f.categories {
		checked := "x"
		if state.UI.Filter != nil && !state.UI.Filter[cat] {
			checked = " "
		}

		badge := CategoryStyle(cat).Render(sse.CategoryBadge(cat))
		cursor := "  "
		if i == f.selectedIdx {
			cursor = lipgloss.NewStyle().Bold(true).Foreground(colorCyan).Render("> ")
		}

		line := fmt.Sprintf("%s[%s] %s %s", cursor, checked, badge, string(cat))
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	overlayStyle := lipgloss.NewStyle().
		Width(30).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		overlayStyle.Render(content))
}
