package format

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", now, "now"},
		{"3 seconds ago", now.Add(-3 * time.Second), "3s ago"},
		{"59 seconds ago", now.Add(-59 * time.Second), "59s ago"},
		{"1 minute ago", now.Add(-60 * time.Second), "1m ago"},
		{"2m15s ago", now.Add(-135 * time.Second), "2m15s ago"},
		{"1 hour ago", now.Add(-time.Hour), "1h ago"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2h ago"},
		{"1 day ago", now.Add(-25 * time.Hour), "1d ago"},
		{"future time", now.Add(time.Hour), "now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RelativeTime(tt.t, now)
			if got != tt.want {
				t.Errorf("RelativeTime(%v) = %q, want %q", tt.t, got, tt.want)
			}
		})
	}
}

func TestCost(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "$0.00"},
		{0.0042, "$0.0042"},
		{0.009, "$0.0090"},
		{0.01, "$0.01"},
		{0.5, "$0.50"},
		{1.23, "$1.23"},
		{10.00, "$10.00"},
		{100.567, "$100.57"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := Cost(tt.input)
			if got != tt.want {
				t.Errorf("Cost(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{5, "5"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{999999, "999,999"},
		{1000000, "1.0M"},
		{1200000, "1.2M"},
		{15000000, "15.0M"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := Tokens(tt.input)
			if got != tt.want {
				t.Errorf("Tokens(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		id     string
		maxLen int
		want   string
	}{
		{"ses_01JXYZ", 20, "ses_01JXYZ"},
		{"ses_01JXYZ", 10, "ses_01JXYZ"},
		{"ses_01JXYZ", 8, "ses_01.."},
		{"ses_01JXYZ", 5, "ses.."},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := ShortID(tt.id, tt.maxLen)
			if got != tt.want {
				t.Errorf("ShortID(%q, %d) = %q, want %q", tt.id, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s        string
		maxWidth int
		want     string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello world", 5, "he..."},
		{"hello", 3, "hel"},
		{"hello", 2, "he"},
		{"hello", 1, "h"},
		{"hello", 0, ""},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := Truncate(tt.s, tt.maxWidth)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxWidth, got, tt.want)
			}
		})
	}
}

func TestTimestamp(t *testing.T) {
	ts := time.Date(2026, 4, 3, 14, 5, 9, 0, time.UTC)
	got := Timestamp(ts)
	want := "14:05:09"
	if got != want {
		t.Errorf("Timestamp = %q, want %q", got, want)
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{100 * time.Millisecond, "0.1s"},
		{500 * time.Millisecond, "0.5s"},
		{time.Second, "1.0s"},
		{3200 * time.Millisecond, "3.2s"},
		{30 * time.Second, "30.0s"},
		{65 * time.Second, "1m5s"},
		{125 * time.Second, "2m5s"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := Duration(tt.d)
			if got != tt.want {
				t.Errorf("Duration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
