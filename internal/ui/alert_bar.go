package ui

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/format"
	"github.com/rxbynerd/monocular/internal/model"
)

// AlertBar renders active permission/question/error alerts.
type AlertBar struct {
	width int
}

func NewAlertBar() AlertBar {
	return AlertBar{}
}

func (a *AlertBar) SetWidth(w int) {
	a.width = w
}

func (a AlertBar) View(state *model.DashboardState, now time.Time) string {
	if len(state.Alerts) == 0 {
		return ""
	}

	// Show most recent alert (newest first in the slice)
	alert := state.Alerts[0]
	elapsed := format.RelativeTime(alert.Timestamp, now)
	sessionShort := format.ShortID(alert.SessionID, 12)

	var icon string
	var style lipgloss.Style
	switch alert.Kind {
	case "permission":
		icon = "[!]"
		style = styleAlertPermission
	case "question":
		icon = "[?]"
		style = styleAlertQuestion
	case "error":
		icon = "[X]"
		style = styleAlertError
	default:
		icon = "[*]"
		style = styleAlertPermission
	}

	text := fmt.Sprintf("%s %s in %s (%s)",
		style.Render(icon),
		alert.Title,
		styleDim.Render(sessionShort),
		styleDim.Render(elapsed),
	)

	if len(state.Alerts) > 1 {
		text += styleDim.Render(fmt.Sprintf("  +%d more", len(state.Alerts)-1))
	}

	barStyle := lipgloss.NewStyle().
		Width(a.width).
		Padding(0, 1).
		Background(lipgloss.Color("52")) // dark red background

	return barStyle.Render(text)
}
