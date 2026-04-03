package sse

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// EventCategory groups event types for filtering and color-coding.
type EventCategory string

const (
	CategorySession    EventCategory = "session"
	CategoryMessage    EventCategory = "message"
	CategoryPermission EventCategory = "permission"
	CategoryQuestion   EventCategory = "question"
	CategoryFile       EventCategory = "file"
	CategoryInfra      EventCategory = "infra"
	CategoryPTY        EventCategory = "pty"
	CategoryWorkspace  EventCategory = "workspace"
	CategoryTUI        EventCategory = "tui"
	CategoryTodo       EventCategory = "todo"
)

// infraPrefixes are event type prefixes that map to CategoryInfra
// regardless of the first dot-segment.
var infraPrefixes = map[string]bool{
	"server":       true,
	"installation": true,
	"lsp":          true,
	"mcp":          true,
	"command":      true,
	"global":       true,
}

// filePrefixes are event type prefixes that map to CategoryFile.
var filePrefixes = map[string]bool{
	"vcs":     true,
	"project": true,
}

// workspacePrefixes are event type prefixes that map to CategoryWorkspace.
var workspacePrefixes = map[string]bool{
	"workspace": true,
	"worktree":  true,
}

// Categorize returns the EventCategory for a given event type string.
// It splits on the first '.' and applies special-case mappings for
// infrastructure and file-related prefixes.
func Categorize(eventType string) EventCategory {
	prefix, _, _ := strings.Cut(eventType, ".")
	if prefix == "" {
		return CategoryInfra
	}

	if infraPrefixes[prefix] {
		return CategoryInfra
	}
	if filePrefixes[prefix] {
		return CategoryFile
	}
	if workspacePrefixes[prefix] {
		return CategoryWorkspace
	}

	switch EventCategory(prefix) {
	case CategorySession:
		return CategorySession
	case CategoryMessage:
		return CategoryMessage
	case CategoryPermission:
		return CategoryPermission
	case CategoryQuestion:
		return CategoryQuestion
	case CategoryFile:
		return CategoryFile
	case CategoryPTY:
		return CategoryPTY
	case CategoryWorkspace:
		return CategoryWorkspace
	case CategoryTUI:
		return CategoryTUI
	case CategoryTodo:
		return CategoryTodo
	default:
		return CategoryInfra
	}
}

// CategoryColor returns the lipgloss color for a category.
func CategoryColor(cat EventCategory) color.Color {
	switch cat {
	case CategorySession:
		return lipgloss.Color("6") // cyan
	case CategoryMessage:
		return lipgloss.Color("2") // green
	case CategoryPermission:
		return lipgloss.Color("3") // yellow
	case CategoryQuestion:
		return lipgloss.Color("5") // magenta
	case CategoryFile:
		return lipgloss.Color("4") // blue
	case CategoryInfra:
		return lipgloss.Color("8") // dim gray
	case CategoryPTY:
		return lipgloss.Color("7") // white
	case CategoryWorkspace:
		return lipgloss.Color("6") // cyan (same as session)
	case CategoryTUI:
		return lipgloss.Color("5") // magenta (same as question)
	case CategoryTodo:
		return lipgloss.Color("2") // green (same as message)
	default:
		return lipgloss.Color("8")
	}
}

// CategoryBadge returns the short bracketed label for display in the event log.
func CategoryBadge(cat EventCategory) string {
	switch cat {
	case CategorySession:
		return "[session]"
	case CategoryMessage:
		return "[message]"
	case CategoryPermission:
		return "[perm]"
	case CategoryQuestion:
		return "[question]"
	case CategoryFile:
		return "[file]"
	case CategoryInfra:
		return "[infra]"
	case CategoryPTY:
		return "[pty]"
	case CategoryWorkspace:
		return "[workspace]"
	case CategoryTUI:
		return "[tui]"
	case CategoryTodo:
		return "[todo]"
	default:
		return "[unknown]"
	}
}

// AllCategories returns all event categories for the filter picker.
func AllCategories() []EventCategory {
	return []EventCategory{
		CategorySession,
		CategoryMessage,
		CategoryPermission,
		CategoryQuestion,
		CategoryFile,
		CategoryInfra,
		CategoryPTY,
		CategoryWorkspace,
		CategoryTUI,
		CategoryTodo,
	}
}
