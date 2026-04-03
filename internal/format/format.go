package format

import (
	"fmt"
	"math"
	"time"
)

// RelativeTime returns a human-readable relative time string.
func RelativeTime(t time.Time, now time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Second:
		return "now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm ago", m)
		}
		return fmt.Sprintf("%dm%ds ago", m, s)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(d.Hours()) / 24
		return fmt.Sprintf("%dd ago", days)
	}
}

// Cost formats a dollar cost value for display.
func Cost(c float64) string {
	if c == 0 {
		return "$0.00"
	}
	if c < 0.01 {
		return fmt.Sprintf("$%.4f", c)
	}
	return fmt.Sprintf("$%.2f", c)
}

// Tokens formats a token count for compact display.
func Tokens(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return formatWithCommas(n)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
}

// ShortID truncates an ID to maxLen characters with ".." suffix.
func ShortID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	if maxLen <= 2 {
		return id[:maxLen]
	}
	return id[:maxLen-2] + ".."
}

// Timestamp formats a time as HH:MM:SS for the event log.
func Timestamp(t time.Time) string {
	return t.Format("15:04:05")
}

// Duration formats a duration for display (tool elapsed times).
func Duration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(math.Mod(d.Seconds(), 60))
	return fmt.Sprintf("%dm%ds", m, s)
}

func formatWithCommas(n int) string {
	if n < 0 {
		return "-" + formatWithCommas(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatWithCommas(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}
