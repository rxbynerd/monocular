package model

import (
	"time"

	"github.com/rxbynerd/monocular/internal/sse"
)

// DashboardState holds all state for the Monocular dashboard.
type DashboardState struct {
	Connection   ConnectionInfo
	Sessions     map[string]*SessionEntry
	SessionOrder []string // Ordered by last activity (most recent first)
	Events       []EventLogEntry
	MaxEvents    int // Default: 500
	Alerts       []Alert
	Counters     Counters
	UI           UIState
}

// NewDashboardState creates a new DashboardState with sensible defaults.
func NewDashboardState() *DashboardState {
	return &DashboardState{
		Connection: ConnectionInfo{
			State: sse.Disconnected,
		},
		Sessions:  make(map[string]*SessionEntry),
		MaxEvents: 500,
		Counters: Counters{
			EventsByType: make(map[string]int),
			FilesEdited:  make(map[string]struct{}),
		},
		UI: UIState{
			FocusedPanel: PanelEvents,
		},
	}
}

type ConnectionInfo struct {
	State          sse.ConnectionState
	URL            string
	ConnectedAt    time.Time
	ReconnectCount int
	LastEventAt    time.Time
}

type SessionEntry struct {
	ID           string
	Slug         string
	Title        string
	Directory    string
	Status       SessionStatus
	LastActivity time.Time
	ActiveTools  []ToolExecution
	MessageCount int
	TotalCost    float64
	TotalTokens  TokenCounts
}

type SessionStatus struct {
	Type    string // "idle", "busy", "retry"
	Attempt int    // only for retry
	Message string // only for retry
}

type ToolExecution struct {
	CallID    string
	Tool      string
	SessionID string
	Status    string // "pending" or "running"
	StartedAt time.Time
}

type TokenCounts struct {
	Input     int
	Output    int
	Reasoning int
}

type EventLogEntry struct {
	ID         int
	Timestamp  time.Time
	Directory  string
	Type       string
	Category   sse.EventCategory
	Summary    string
	Properties map[string]any
}

type Alert struct {
	ID         string
	Kind       string // "permission", "question", "error"
	SessionID  string
	Title      string
	Timestamp  time.Time
	Properties map[string]any
}

type Counters struct {
	TotalEvents  int
	EventsByType map[string]int
	TotalCost    float64
	TotalTokens  TokenCounts
	FilesEdited  map[string]struct{}
}

type UIState struct {
	SelectedSessionID  string
	Directory          string
	SelectedEventIdx   int
	Filter             map[sse.EventCategory]bool
	DetailExpanded     bool
	FocusedPanel       Panel
	Paused             bool
	PauseBuffer        []EventLogEntry
	SearchQuery        string
	SearchActive       bool
	ShowHelp           bool
	ShowFilter         bool
	StreamingIndicator map[string]time.Time // sessionID -> last delta time
}

type Panel int

const (
	PanelSessions Panel = iota
	PanelEvents
	PanelDetail
)

func (p Panel) String() string {
	switch p {
	case PanelSessions:
		return "Sessions"
	case PanelEvents:
		return "Events"
	case PanelDetail:
		return "Detail"
	default:
		return "Unknown"
	}
}
