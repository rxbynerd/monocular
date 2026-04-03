package ui

import (
	"charm.land/lipgloss/v2"
)

// HelpBar renders the bottom bar with keybinding hints.
type HelpBar struct {
	width int
}

func NewHelpBar() HelpBar {
	return HelpBar{}
}

func (b *HelpBar) SetWidth(w int) {
	b.width = w
}

type helpEntry struct {
	key  string
	desc string
}

var helpEntries = []helpEntry{
	{"q", "quit"},
	{"tab", "panel"},
	{"j/k", "scroll"},
	{"enter", "expand"},
	{"f", "filter"},
	{"/", "search"},
	{"p", "pause"},
	{"c", "clear"},
	{"?", "help"},
}

func (b HelpBar) View() string {
	content := ""
	for i, e := range helpEntries {
		if i > 0 {
			content += "  "
		}
		content += styleHelpKey.Render(e.key) + ":" + styleHelpDesc.Render(e.desc)
	}

	style := lipgloss.NewStyle().
		Width(b.width).
		Padding(0, 1).
		Background(lipgloss.Color("235"))

	return style.Render(content)
}
