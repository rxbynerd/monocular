package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/sse"
)

// Colors
var (
	colorCyan    = lipgloss.Color("6")
	colorGreen   = lipgloss.Color("2")
	colorYellow  = lipgloss.Color("3")
	colorMagenta = lipgloss.Color("5")
	colorBlue    = lipgloss.Color("4")
	colorGray    = lipgloss.Color("8")
	colorWhite   = lipgloss.Color("7")
	colorRed     = lipgloss.Color("1")
	colorDim     = lipgloss.Color("240")
	colorBright  = lipgloss.Color("15")
)

// Panel borders
var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBright)

	unfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim)
)

// Connection states
var (
	styleConnected    = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	styleConnecting   = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	styleDisconnected = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	styleConnectFailed = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
)

// Session status badges
var (
	badgeIdle  = lipgloss.NewStyle().Foreground(colorGreen).Render("[IDLE]")
	badgeBusy  = lipgloss.NewStyle().Bold(true).Foreground(colorYellow).Render("[BUSY]")
	badgeRetry = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("[RETRY]")
)

// Tool status styles
var (
	styleToolPending   = lipgloss.NewStyle().Foreground(colorDim)
	styleToolRunning   = lipgloss.NewStyle().Foreground(colorYellow)
	styleToolCompleted = lipgloss.NewStyle().Foreground(colorGreen)
	styleToolError     = lipgloss.NewStyle().Foreground(colorRed)
)

// Alert styles
var (
	styleAlertPermission = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	styleAlertQuestion   = lipgloss.NewStyle().Bold(true).Foreground(colorMagenta)
	styleAlertError      = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
)

// General
var (
	styleTitle    = lipgloss.NewStyle().Bold(true)
	styleDim      = lipgloss.NewStyle().Foreground(colorDim)
	styleHelpKey  = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colorGray)
)

// SessionStatusBadge returns the styled badge string for a session status.
func SessionStatusBadge(statusType string) string {
	switch statusType {
	case "busy":
		return badgeBusy
	case "retry":
		return badgeRetry
	default:
		return badgeIdle
	}
}

// CategoryStyle returns a lipgloss style for an event category badge.
func CategoryStyle(cat sse.EventCategory) lipgloss.Style {
	color := sse.CategoryColor(cat)
	s := lipgloss.NewStyle().Foreground(color)
	if cat == sse.CategoryPermission {
		s = s.Bold(true)
	}
	return s
}

// PanelStyle returns the border style for a panel based on focus state.
func PanelStyle(focused bool, width, height int) lipgloss.Style {
	if focused {
		return focusedBorder.Width(width).Height(height)
	}
	return unfocusedBorder.Width(width).Height(height)
}
