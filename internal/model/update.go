package model

import (
	"fmt"
	"time"

	"github.com/rxbynerd/monocular/internal/format"
	"github.com/rxbynerd/monocular/internal/sse"
)

// ApplyEvent processes an SSE event and mutates the dashboard state.
// now is the current time, passed in for testability.
func ApplyEvent(s *DashboardState, event sse.GlobalEvent, now time.Time) {
	eventType := event.Payload.Type
	props := event.Payload.Properties

	s.Connection.LastEventAt = now

	// Skip heartbeat from event log but still update LastEventAt
	if eventType == "server.heartbeat" {
		return
	}

	// Skip message.part.delta from event log; update streaming indicator
	if eventType == "message.part.delta" {
		sessionID := getString(props, "sessionID")
		if sessionID != "" {
			if s.UI.StreamingIndicator == nil {
				s.UI.StreamingIndicator = make(map[string]time.Time)
			}
			s.UI.StreamingIndicator[sessionID] = now
		}
		return
	}

	// Apply state mutations based on event type
	switch eventType {
	case "server.connected":
		s.Connection.State = sse.Connected
		s.Connection.ConnectedAt = now

	case "session.created":
		applySessionCreated(s, props, event.Directory, now)
	case "session.updated":
		applySessionUpdated(s, props, now)
	case "session.deleted":
		applySessionDeleted(s, props)
	case "session.status":
		applySessionStatus(s, props, now)
	case "session.idle":
		applySessionIdle(s, props, now)
	case "session.error":
		applySessionError(s, props, now)

	case "message.updated":
		applyMessageUpdated(s, props, now)
	case "message.part.updated":
		applyMessagePartUpdated(s, props, now)

	case "permission.asked":
		applyPermissionAsked(s, props, now)
	case "permission.replied":
		removeAlert(s, getString(props, "id"))

	case "question.asked":
		applyQuestionAsked(s, props, now)
	case "question.replied", "question.rejected":
		removeAlert(s, getString(props, "id"))

	case "file.edited":
		file := getString(props, "file")
		if file != "" {
			s.Counters.FilesEdited[file] = struct{}{}
		}

	case "_reconnected":
		s.Connection.ReconnectCount++
	}

	// Add to event log (for all non-skipped events)
	entry := EventLogEntry{
		ID:         s.Counters.TotalEvents + 1,
		Timestamp:  now,
		Directory:  event.Directory,
		Type:       eventType,
		Category:   sse.Categorize(eventType),
		Summary:    generateSummary(s, eventType, props),
		Properties: props,
	}

	s.Counters.TotalEvents++
	s.Counters.EventsByType[eventType]++

	if s.UI.Paused {
		s.UI.PauseBuffer = append(s.UI.PauseBuffer, entry)
		if len(s.UI.PauseBuffer) > s.MaxEvents {
			s.UI.PauseBuffer = s.UI.PauseBuffer[1:]
		}
	} else {
		s.Events = append(s.Events, entry)
		if len(s.Events) > s.MaxEvents {
			s.Events = s.Events[1:]
		}
	}
}

// ApplySessionsRefreshed updates session state from REST data after reconnect.
func ApplySessionsRefreshed(s *DashboardState, sessions []sse.SessionInfo, now time.Time) {
	seen := make(map[string]bool)
	for _, info := range sessions {
		seen[info.ID] = true
		entry, exists := s.Sessions[info.ID]
		if !exists {
			entry = &SessionEntry{
				ID:     info.ID,
				Status: SessionStatus{Type: "idle"},
			}
			s.Sessions[info.ID] = entry
			s.SessionOrder = append([]string{info.ID}, s.SessionOrder...)
		}
		entry.Slug = info.Slug
		entry.Title = info.Title
		entry.Directory = info.Directory
		entry.LastActivity = now
	}
}

// FlushPauseBuffer moves buffered events into the main event log.
func FlushPauseBuffer(s *DashboardState) {
	s.Events = append(s.Events, s.UI.PauseBuffer...)
	if len(s.Events) > s.MaxEvents {
		s.Events = s.Events[len(s.Events)-s.MaxEvents:]
	}
	s.UI.PauseBuffer = nil
}

// ClearStreamingIndicators removes stale streaming indicators.
func ClearStreamingIndicators(s *DashboardState, now time.Time, timeout time.Duration) {
	for sid, lastDelta := range s.UI.StreamingIndicator {
		if now.Sub(lastDelta) > timeout {
			delete(s.UI.StreamingIndicator, sid)
		}
	}
}

func applySessionCreated(s *DashboardState, props map[string]any, dir string, now time.Time) {
	sessionID := getString(props, "sessionID")
	if sessionID == "" {
		return
	}

	info := getMap(props, "info")
	entry := &SessionEntry{
		ID:           sessionID,
		Slug:         getString(info, "slug"),
		Title:        getString(info, "title"),
		Directory:    dir,
		Status:       SessionStatus{Type: "idle"},
		LastActivity: now,
	}
	if d := getString(info, "directory"); d != "" {
		entry.Directory = d
	}

	s.Sessions[sessionID] = entry
	s.SessionOrder = append([]string{sessionID}, s.SessionOrder...)
}

func applySessionUpdated(s *DashboardState, props map[string]any, now time.Time) {
	sessionID := getString(props, "sessionID")
	entry, ok := s.Sessions[sessionID]
	if !ok {
		return
	}

	info := getMap(props, "info")
	if title := getString(info, "title"); title != "" {
		entry.Title = title
	}
	if slug := getString(info, "slug"); slug != "" {
		entry.Slug = slug
	}
	entry.LastActivity = now
	bumpSession(s, sessionID)
}

func applySessionDeleted(s *DashboardState, props map[string]any) {
	sessionID := getString(props, "sessionID")
	if sessionID == "" {
		// Fallback: extract from info.id
		info := getMap(props, "info")
		sessionID = getString(info, "id")
	}
	if sessionID == "" {
		return
	}

	delete(s.Sessions, sessionID)
	for i, id := range s.SessionOrder {
		if id == sessionID {
			s.SessionOrder = append(s.SessionOrder[:i], s.SessionOrder[i+1:]...)
			break
		}
	}
}

func applySessionStatus(s *DashboardState, props map[string]any, now time.Time) {
	sessionID := getString(props, "sessionID")
	entry, ok := s.Sessions[sessionID]
	if !ok {
		return
	}

	statusMap := getMap(props, "status")
	statusType := getString(statusMap, "type")

	entry.Status = SessionStatus{Type: statusType}
	if statusType == "retry" {
		entry.Status.Attempt = getInt(statusMap, "attempt")
		entry.Status.Message = getString(statusMap, "message")
	}
	entry.LastActivity = now
	bumpSession(s, sessionID)
}

func applySessionIdle(s *DashboardState, props map[string]any, now time.Time) {
	sessionID := getString(props, "sessionID")
	entry, ok := s.Sessions[sessionID]
	if !ok {
		return
	}
	entry.Status = SessionStatus{Type: "idle"}
	entry.ActiveTools = nil
	entry.LastActivity = now
	bumpSession(s, sessionID)
}

func applySessionError(s *DashboardState, props map[string]any, now time.Time) {
	sessionID := getString(props, "sessionID")
	errMsg := getString(props, "error")

	alert := Alert{
		ID:         fmt.Sprintf("error-%s-%d", sessionID, now.UnixNano()),
		Kind:       "error",
		SessionID:  sessionID,
		Title:      fmt.Sprintf("Session error: %s", format.Truncate(errMsg, 60)),
		Timestamp:  now,
		Properties: props,
	}
	s.Alerts = append([]Alert{alert}, s.Alerts...)
}

func applyMessageUpdated(s *DashboardState, props map[string]any, now time.Time) {
	sessionID := getString(props, "sessionID")
	entry, ok := s.Sessions[sessionID]
	if !ok {
		return
	}

	entry.MessageCount++
	entry.LastActivity = now
	bumpSession(s, sessionID)

	info := getMap(props, "info")
	role := getString(info, "role")
	if role == "assistant" {
		cost := getFloat(info, "cost")
		if cost > 0 {
			entry.TotalCost += cost
			s.Counters.TotalCost += cost
		}

		tokens := getMap(info, "tokens")
		if tokens != nil {
			input := getInt(tokens, "input")
			output := getInt(tokens, "output")
			reasoning := getInt(tokens, "reasoning")

			entry.TotalTokens.Input += input
			entry.TotalTokens.Output += output
			entry.TotalTokens.Reasoning += reasoning

			s.Counters.TotalTokens.Input += input
			s.Counters.TotalTokens.Output += output
			s.Counters.TotalTokens.Reasoning += reasoning
		}
	}
}

func applyMessagePartUpdated(s *DashboardState, props map[string]any, now time.Time) {
	sessionID := getString(props, "sessionID")
	entry, ok := s.Sessions[sessionID]
	if !ok {
		return
	}

	part := getMap(props, "part")
	partType := getString(part, "type")
	if partType != "tool" {
		return
	}

	toolID := getString(part, "toolID")
	callID := getString(part, "id")
	state := getMap(part, "state")
	status := getString(state, "status")

	switch status {
	case "pending", "running":
		// Add or update tool execution
		found := false
		for i := range entry.ActiveTools {
			if entry.ActiveTools[i].CallID == callID {
				entry.ActiveTools[i].Status = status
				found = true
				break
			}
		}
		if !found {
			entry.ActiveTools = append(entry.ActiveTools, ToolExecution{
				CallID:    callID,
				Tool:      toolID,
				SessionID: sessionID,
				Status:    status,
				StartedAt: now,
			})
		}
	case "completed", "error":
		// Remove from active tools
		for i := range entry.ActiveTools {
			if entry.ActiveTools[i].CallID == callID {
				entry.ActiveTools = append(entry.ActiveTools[:i], entry.ActiveTools[i+1:]...)
				break
			}
		}
	}

	entry.LastActivity = now
	bumpSession(s, sessionID)
}

func applyPermissionAsked(s *DashboardState, props map[string]any, now time.Time) {
	id := getString(props, "id")
	sessionID := getString(props, "sessionID")
	permission := getString(props, "permission")

	title := fmt.Sprintf("Permission requested: %s", permission)
	patterns := getSlice(props, "patterns")
	if len(patterns) > 0 {
		if p, ok := patterns[0].(string); ok {
			title += fmt.Sprintf(" (%s)", format.Truncate(p, 40))
		}
	}

	alert := Alert{
		ID:         id,
		Kind:       "permission",
		SessionID:  sessionID,
		Title:      title,
		Timestamp:  now,
		Properties: props,
	}
	s.Alerts = append([]Alert{alert}, s.Alerts...)
}

func applyQuestionAsked(s *DashboardState, props map[string]any, now time.Time) {
	id := getString(props, "id")
	sessionID := getString(props, "sessionID")
	title := getString(props, "title")

	alert := Alert{
		ID:         id,
		Kind:       "question",
		SessionID:  sessionID,
		Title:      fmt.Sprintf("Question: %s", format.Truncate(title, 60)),
		Timestamp:  now,
		Properties: props,
	}
	s.Alerts = append([]Alert{alert}, s.Alerts...)
}

func removeAlert(s *DashboardState, id string) {
	for i, a := range s.Alerts {
		if a.ID == id {
			s.Alerts = append(s.Alerts[:i], s.Alerts[i+1:]...)
			return
		}
	}
}

// bumpSession moves a session to the front of SessionOrder.
func bumpSession(s *DashboardState, sessionID string) {
	for i, id := range s.SessionOrder {
		if id == sessionID {
			if i == 0 {
				return
			}
			s.SessionOrder = append(s.SessionOrder[:i], s.SessionOrder[i+1:]...)
			s.SessionOrder = append([]string{sessionID}, s.SessionOrder...)
			return
		}
	}
}

func generateSummary(s *DashboardState, eventType string, props map[string]any) string {
	switch eventType {
	case "session.created":
		info := getMap(props, "info")
		return fmt.Sprintf("Session created: %s", getString(info, "title"))
	case "session.updated":
		info := getMap(props, "info")
		return fmt.Sprintf("Session updated: %s", getString(info, "title"))
	case "session.deleted":
		info := getMap(props, "info")
		return fmt.Sprintf("Session deleted: %s", getString(info, "title"))
	case "session.status":
		slug := getString(props, "slug")
		statusMap := getMap(props, "status")
		return fmt.Sprintf("Session %s: %s", slug, getString(statusMap, "type"))
	case "session.idle":
		sessionID := getString(props, "sessionID")
		if entry, ok := s.Sessions[sessionID]; ok {
			return fmt.Sprintf("Session %s: idle", entry.Slug)
		}
		return "Session idle"
	case "session.error":
		return fmt.Sprintf("Session error: %s", format.Truncate(getString(props, "error"), 50))

	case "message.updated":
		info := getMap(props, "info")
		role := getString(info, "role")
		sessionID := getString(props, "sessionID")
		slug := ""
		if entry, ok := s.Sessions[sessionID]; ok {
			slug = entry.Slug
		}
		summary := fmt.Sprintf("Message from %s in %s", role, slug)
		cost := getFloat(info, "cost")
		if cost > 0 {
			summary += fmt.Sprintf(" (%s)", format.Cost(cost))
		}
		return summary
	case "message.part.updated":
		part := getMap(props, "part")
		partType := getString(part, "type")
		if partType == "tool" {
			toolID := getString(part, "toolID")
			state := getMap(part, "state")
			status := getString(state, "status")
			return fmt.Sprintf("[tool] %s %s", toolID, status)
		}
		return fmt.Sprintf("[%s]", partType)

	case "permission.asked":
		perm := getString(props, "permission")
		patterns := getSlice(props, "patterns")
		summary := fmt.Sprintf("Permission requested: %s", perm)
		if len(patterns) > 0 {
			if p, ok := patterns[0].(string); ok {
				summary += fmt.Sprintf(" %s", format.Truncate(p, 30))
			}
		}
		return summary

	case "file.edited":
		return fmt.Sprintf("File edited: %s", getString(props, "file"))

	case "_reconnected":
		return "--- reconnected ---"

	default:
		return eventType
	}
}

// Helper functions for safe property extraction from map[string]any.

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func getFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

func getInt(m map[string]any, key string) int {
	return int(getFloat(m, key))
}

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	sub, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return sub
}

func getSlice(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}
