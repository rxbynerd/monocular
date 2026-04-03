package model

import (
	"testing"
	"time"

	"github.com/rxbynerd/monocular/internal/sse"
	"github.com/rxbynerd/monocular/testdata"
)

var now = testdata.Now

func newState() *DashboardState {
	return NewDashboardState()
}

func TestSessionCreated(t *testing.T) {
	s := newState()
	ev := testdata.SessionCreated("ses_01", "fix-bug", "Fix auth bug", "/proj")
	ApplyEvent(s, ev, now)

	if len(s.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(s.Sessions))
	}
	entry := s.Sessions["ses_01"]
	if entry.ID != "ses_01" {
		t.Errorf("ID = %q, want ses_01", entry.ID)
	}
	if entry.Slug != "fix-bug" {
		t.Errorf("Slug = %q, want fix-bug", entry.Slug)
	}
	if entry.Title != "Fix auth bug" {
		t.Errorf("Title = %q, want Fix auth bug", entry.Title)
	}
	if entry.Status.Type != "idle" {
		t.Errorf("Status.Type = %q, want idle", entry.Status.Type)
	}
	if len(s.SessionOrder) != 1 || s.SessionOrder[0] != "ses_01" {
		t.Errorf("SessionOrder = %v, want [ses_01]", s.SessionOrder)
	}
	if s.Counters.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1", s.Counters.TotalEvents)
	}
}

func TestSessionStatusBusy(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix-bug", "Fix auth bug", "/proj"), now)
	ApplyEvent(s, testdata.SessionStatus("ses_01", "fix-bug", "busy"), now)

	entry := s.Sessions["ses_01"]
	if entry.Status.Type != "busy" {
		t.Errorf("Status.Type = %q, want busy", entry.Status.Type)
	}
}

func TestSessionStatusRetry(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix-bug", "Fix", "/proj"), now)
	ApplyEvent(s, testdata.SessionStatusRetry("ses_01", "fix-bug", 3, "rate limited"), now)

	entry := s.Sessions["ses_01"]
	if entry.Status.Type != "retry" {
		t.Errorf("Status.Type = %q, want retry", entry.Status.Type)
	}
	if entry.Status.Attempt != 3 {
		t.Errorf("Attempt = %d, want 3", entry.Status.Attempt)
	}
	if entry.Status.Message != "rate limited" {
		t.Errorf("Message = %q, want 'rate limited'", entry.Status.Message)
	}
}

func TestSessionStatusIdle(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix-bug", "Fix", "/proj"), now)
	ApplyEvent(s, testdata.SessionStatus("ses_01", "fix-bug", "busy"), now)

	// Add a tool to verify it gets cleared on idle
	s.Sessions["ses_01"].ActiveTools = []ToolExecution{
		{CallID: "call_01", Tool: "bash", SessionID: "ses_01"},
	}

	ApplyEvent(s, testdata.SessionIdle("ses_01"), now)

	entry := s.Sessions["ses_01"]
	if entry.Status.Type != "idle" {
		t.Errorf("Status.Type = %q, want idle", entry.Status.Type)
	}
	if len(entry.ActiveTools) != 0 {
		t.Errorf("ActiveTools should be cleared on idle, got %d", len(entry.ActiveTools))
	}
}

func TestSessionDeleted(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix-bug", "Fix", "/proj"), now)
	ApplyEvent(s, testdata.SessionDeleted("ses_01", "fix-bug", "Fix", "/proj"), now)

	if len(s.Sessions) != 0 {
		t.Errorf("expected 0 sessions after delete, got %d", len(s.Sessions))
	}
	if len(s.SessionOrder) != 0 {
		t.Errorf("SessionOrder should be empty, got %v", s.SessionOrder)
	}
}

func TestSessionDeletedLegacy(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix-bug", "Fix", "/proj"), now)
	ApplyEvent(s, testdata.SessionDeletedLegacy("ses_01", "fix-bug", "Fix", "/proj"), now)

	if len(s.Sessions) != 0 {
		t.Errorf("expected 0 sessions after legacy delete, got %d", len(s.Sessions))
	}
}

func TestFullSessionLifecycle(t *testing.T) {
	s := newState()
	events := testdata.SessionLifecycle("ses_01", "fix-bug", "Fix auth bug", "/proj")
	for _, ev := range events {
		ApplyEvent(s, ev, now)
	}

	if len(s.Sessions) != 0 {
		t.Errorf("session should be deleted, got %d sessions", len(s.Sessions))
	}
	if s.Counters.TotalEvents != 4 {
		t.Errorf("TotalEvents = %d, want 4", s.Counters.TotalEvents)
	}
}

func TestToolLifecycle(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix-bug", "Fix", "/proj"), now)

	events := testdata.ToolLifecycle("ses_01", "bash", "call_01")

	// pending
	ApplyEvent(s, events[0], now)
	entry := s.Sessions["ses_01"]
	if len(entry.ActiveTools) != 1 {
		t.Fatalf("expected 1 active tool after pending, got %d", len(entry.ActiveTools))
	}
	if entry.ActiveTools[0].Status != "pending" {
		t.Errorf("tool status = %q, want pending", entry.ActiveTools[0].Status)
	}

	// running
	ApplyEvent(s, events[1], now)
	if entry.ActiveTools[0].Status != "running" {
		t.Errorf("tool status = %q, want running", entry.ActiveTools[0].Status)
	}

	// completed
	ApplyEvent(s, events[2], now)
	if len(entry.ActiveTools) != 0 {
		t.Errorf("expected 0 active tools after completion, got %d", len(entry.ActiveTools))
	}
}

func TestToolError(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix", "Fix", "/p"), now)
	ApplyEvent(s, testdata.MessagePartTool("ses_01", "bash", "c1", "running"), now)
	ApplyEvent(s, testdata.MessagePartTool("ses_01", "bash", "c1", "error"), now)

	entry := s.Sessions["ses_01"]
	if len(entry.ActiveTools) != 0 {
		t.Errorf("tool should be removed on error, got %d active tools", len(entry.ActiveTools))
	}
}

func TestPermissionLifecycle(t *testing.T) {
	s := newState()
	events := testdata.PermissionLifecycle("perm_01", "ses_01", "bash")

	ApplyEvent(s, events[0], now) // asked
	if len(s.Alerts) != 1 {
		t.Fatalf("expected 1 alert after permission.asked, got %d", len(s.Alerts))
	}
	if s.Alerts[0].Kind != "permission" {
		t.Errorf("alert kind = %q, want permission", s.Alerts[0].Kind)
	}
	if s.Alerts[0].ID != "perm_01" {
		t.Errorf("alert ID = %q, want perm_01", s.Alerts[0].ID)
	}

	ApplyEvent(s, events[1], now) // replied
	if len(s.Alerts) != 0 {
		t.Errorf("expected 0 alerts after permission.replied, got %d", len(s.Alerts))
	}
}

func TestQuestionLifecycle(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SampleEvents["question.asked"], now)

	if len(s.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(s.Alerts))
	}
	if s.Alerts[0].Kind != "question" {
		t.Errorf("alert kind = %q, want question", s.Alerts[0].Kind)
	}

	ApplyEvent(s, testdata.SampleEvents["question.replied"], now)
	if len(s.Alerts) != 0 {
		t.Errorf("expected 0 alerts after reply, got %d", len(s.Alerts))
	}
}

func TestQuestionRejected(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SampleEvents["question.asked"], now)
	ApplyEvent(s, testdata.SampleEvents["question.rejected"], now)

	if len(s.Alerts) != 0 {
		t.Errorf("expected 0 alerts after rejection, got %d", len(s.Alerts))
	}
}

func TestHeartbeatDoesNotAddToLog(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SampleEvents["server.heartbeat"], now)

	if len(s.Events) != 0 {
		t.Errorf("heartbeat should not add to event log, got %d events", len(s.Events))
	}
	if s.Counters.TotalEvents != 0 {
		t.Errorf("heartbeat should not increment counter, got %d", s.Counters.TotalEvents)
	}
	if !s.Connection.LastEventAt.Equal(now) {
		t.Error("heartbeat should update LastEventAt")
	}
}

func TestDeltaSkippedFromLog(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SampleEvents["message.part.delta"], now)

	if len(s.Events) != 0 {
		t.Errorf("delta should not add to event log, got %d events", len(s.Events))
	}
	if s.UI.StreamingIndicator == nil {
		t.Fatal("streaming indicator should be set")
	}
	if _, ok := s.UI.StreamingIndicator["ses_01JTEST"]; !ok {
		t.Error("streaming indicator should be set for session")
	}
}

func TestRingBufferOverflow(t *testing.T) {
	s := newState()
	s.MaxEvents = 10

	for i := 0; i < 15; i++ {
		ev := testdata.FileEdited("/proj", "file.go")
		ApplyEvent(s, ev, now)
	}

	if len(s.Events) != 10 {
		t.Errorf("expected 10 events (ring buffer), got %d", len(s.Events))
	}
	// First event should be ID 6 (dropped 1-5)
	if s.Events[0].ID != 6 {
		t.Errorf("first event ID = %d, want 6", s.Events[0].ID)
	}
}

func TestCostAccumulation(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix", "Fix", "/p"), now)

	tokens := map[string]any{"input": float64(100), "output": float64(50), "reasoning": float64(10)}
	ApplyEvent(s, testdata.MessageUpdated("ses_01", "assistant", 0.005, tokens), now)
	ApplyEvent(s, testdata.MessageUpdated("ses_01", "assistant", 0.003, tokens), now)

	entry := s.Sessions["ses_01"]
	if entry.TotalCost != 0.008 {
		t.Errorf("session cost = %f, want 0.008", entry.TotalCost)
	}
	if s.Counters.TotalCost != 0.008 {
		t.Errorf("global cost = %f, want 0.008", s.Counters.TotalCost)
	}
	if entry.TotalTokens.Input != 200 {
		t.Errorf("session input tokens = %d, want 200", entry.TotalTokens.Input)
	}
	if s.Counters.TotalTokens.Input != 200 {
		t.Errorf("global input tokens = %d, want 200", s.Counters.TotalTokens.Input)
	}
}

func TestUserMessageNoCost(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "fix", "Fix", "/p"), now)
	ApplyEvent(s, testdata.MessageUpdated("ses_01", "user", 0, nil), now)

	if s.Counters.TotalCost != 0 {
		t.Errorf("user message should not add cost, got %f", s.Counters.TotalCost)
	}
}

func TestFileEdited(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.FileEdited("/proj", "src/auth.ts"), now)
	ApplyEvent(s, testdata.FileEdited("/proj", "src/auth.ts"), now) // duplicate
	ApplyEvent(s, testdata.FileEdited("/proj", "src/index.ts"), now)

	if len(s.Counters.FilesEdited) != 2 {
		t.Errorf("expected 2 unique files, got %d", len(s.Counters.FilesEdited))
	}
}

func TestReconnectedEvent(t *testing.T) {
	s := newState()
	ev := sse.GlobalEvent{
		Payload: sse.EventPayload{
			Type:       "_reconnected",
			Properties: map[string]any{},
		},
	}
	ApplyEvent(s, ev, now)

	if s.Connection.ReconnectCount != 1 {
		t.Errorf("ReconnectCount = %d, want 1", s.Connection.ReconnectCount)
	}
	if len(s.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.Events))
	}
	if s.Events[0].Summary != "--- reconnected ---" {
		t.Errorf("summary = %q, want gap marker", s.Events[0].Summary)
	}
}

func TestSessionsRefreshed(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "old-slug", "Old Title", "/proj"), now)

	sessions := []sse.SessionInfo{
		{ID: "ses_01", Slug: "new-slug", Title: "New Title", Directory: "/proj"},
		{ID: "ses_02", Slug: "new-session", Title: "New Session", Directory: "/proj2"},
	}
	ApplySessionsRefreshed(s, sessions, now)

	if len(s.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(s.Sessions))
	}
	if s.Sessions["ses_01"].Slug != "new-slug" {
		t.Errorf("existing session slug should be updated to 'new-slug', got %q", s.Sessions["ses_01"].Slug)
	}
	if s.Sessions["ses_02"].Title != "New Session" {
		t.Errorf("new session title = %q, want 'New Session'", s.Sessions["ses_02"].Title)
	}
}

func TestSessionBumping(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SessionCreated("ses_01", "first", "First", "/p"), now)
	ApplyEvent(s, testdata.SessionCreated("ses_02", "second", "Second", "/p"), now)
	ApplyEvent(s, testdata.SessionCreated("ses_03", "third", "Third", "/p"), now)

	// Order should be: ses_03, ses_02, ses_01 (most recent first)
	if s.SessionOrder[0] != "ses_03" {
		t.Errorf("expected ses_03 first, got %v", s.SessionOrder)
	}

	// Activity on ses_01 should bump it to front
	ApplyEvent(s, testdata.SessionStatus("ses_01", "first", "busy"), now)
	if s.SessionOrder[0] != "ses_01" {
		t.Errorf("expected ses_01 first after bump, got %v", s.SessionOrder)
	}
}

func TestServerConnected(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SampleEvents["server.connected"], now)

	if s.Connection.State != sse.Connected {
		t.Errorf("state = %v, want Connected", s.Connection.State)
	}
	if !s.Connection.ConnectedAt.Equal(now) {
		t.Error("ConnectedAt should be set")
	}
}

func TestPauseBuffer(t *testing.T) {
	s := newState()
	s.UI.Paused = true

	ApplyEvent(s, testdata.FileEdited("/p", "a.go"), now)
	ApplyEvent(s, testdata.FileEdited("/p", "b.go"), now)

	if len(s.Events) != 0 {
		t.Errorf("events should not be added to main log while paused, got %d", len(s.Events))
	}
	if len(s.UI.PauseBuffer) != 2 {
		t.Errorf("pause buffer should have 2 events, got %d", len(s.UI.PauseBuffer))
	}

	s.UI.Paused = false
	FlushPauseBuffer(s)

	if len(s.Events) != 2 {
		t.Errorf("expected 2 events after flush, got %d", len(s.Events))
	}
	if len(s.UI.PauseBuffer) != 0 {
		t.Errorf("pause buffer should be empty after flush, got %d", len(s.UI.PauseBuffer))
	}
}

func TestClearStreamingIndicators(t *testing.T) {
	s := newState()
	s.UI.StreamingIndicator = map[string]time.Time{
		"ses_01": now.Add(-4 * time.Second),
		"ses_02": now.Add(-1 * time.Second),
	}

	ClearStreamingIndicators(s, now, 3*time.Second)

	if _, ok := s.UI.StreamingIndicator["ses_01"]; ok {
		t.Error("stale indicator should be removed")
	}
	if _, ok := s.UI.StreamingIndicator["ses_02"]; !ok {
		t.Error("recent indicator should be kept")
	}
}

func TestSessionError(t *testing.T) {
	s := newState()
	ApplyEvent(s, testdata.SampleEvents["session.error"], now)

	if len(s.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(s.Alerts))
	}
	if s.Alerts[0].Kind != "error" {
		t.Errorf("alert kind = %q, want error", s.Alerts[0].Kind)
	}
}

func TestSummaryGeneration(t *testing.T) {
	tests := []struct {
		name     string
		event    sse.GlobalEvent
		contains string
	}{
		{"session.created", testdata.SessionCreated("s1", "fix", "Fix bug", "/p"), "Session created: Fix bug"},
		{"session.status", testdata.SessionStatus("s1", "fix", "busy"), "Session fix: busy"},
		{"file.edited", testdata.FileEdited("/p", "auth.ts"), "File edited: auth.ts"},
		{"permission.asked", testdata.PermissionAsked("p1", "s1", "bash", []any{"npm install"}), "Permission requested: bash npm install"},
		{"_reconnected", sse.GlobalEvent{Payload: sse.EventPayload{Type: "_reconnected", Properties: map[string]any{}}}, "--- reconnected ---"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newState()
			if tt.event.Payload.Type == "session.status" {
				ApplyEvent(s, testdata.SessionCreated("s1", "fix", "Fix", "/p"), now)
			}
			ApplyEvent(s, tt.event, now)

			found := false
			for _, ev := range s.Events {
				if ev.Type == tt.event.Payload.Type {
					if ev.Summary == "" {
						t.Errorf("summary for %s is empty", tt.name)
					}
					if tt.contains != "" && ev.Summary != tt.contains {
						t.Errorf("summary = %q, want %q", ev.Summary, tt.contains)
					}
					found = true
					break
				}
			}
			if !found {
				t.Errorf("event %s not found in log", tt.name)
			}
		})
	}
}

func TestPauseBufferOverflow(t *testing.T) {
	s := newState()
	s.MaxEvents = 5
	s.UI.Paused = true

	for i := 0; i < 10; i++ {
		ApplyEvent(s, testdata.FileEdited("/p", "a.go"), now)
	}

	if len(s.UI.PauseBuffer) != 5 {
		t.Errorf("pause buffer should be capped at MaxEvents, got %d", len(s.UI.PauseBuffer))
	}

	s.UI.Paused = false
	FlushPauseBuffer(s)

	if len(s.Events) != 5 {
		t.Errorf("expected 5 events after flush, got %d", len(s.Events))
	}
}

func TestFlushPauseBufferOverflow(t *testing.T) {
	s := newState()
	s.MaxEvents = 5

	// Add 3 events to main log
	for i := 0; i < 3; i++ {
		ApplyEvent(s, testdata.FileEdited("/p", "a.go"), now)
	}

	// Pause and add 4 more
	s.UI.Paused = true
	for i := 0; i < 4; i++ {
		ApplyEvent(s, testdata.FileEdited("/p", "b.go"), now)
	}

	s.UI.Paused = false
	FlushPauseBuffer(s)

	// 3 + 4 = 7, capped to 5
	if len(s.Events) != 5 {
		t.Errorf("expected 5 events (capped), got %d", len(s.Events))
	}
}

func TestAllSampleEventsCanBeApplied(t *testing.T) {
	s := newState()
	// Create a session first so events that reference ses_01JTEST work
	ApplyEvent(s, testdata.SessionCreated("ses_01JTEST", "fix-auth-bug", "Fix auth bug", "/Users/dev/myproject"), now)

	for name, ev := range testdata.SampleEvents {
		t.Run(name, func(t *testing.T) {
			// Should not panic
			ApplyEvent(s, ev, now)
		})
	}
}
