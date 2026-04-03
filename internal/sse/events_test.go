package sse

import (
	"testing"
)

func TestCategorize(t *testing.T) {
	tests := []struct {
		eventType string
		want      EventCategory
	}{
		// Session
		{"session.created", CategorySession},
		{"session.updated", CategorySession},
		{"session.deleted", CategorySession},
		{"session.status", CategorySession},
		{"session.idle", CategorySession},
		{"session.compacted", CategorySession},
		{"session.diff", CategorySession},
		{"session.error", CategorySession},

		// Message
		{"message.updated", CategoryMessage},
		{"message.removed", CategoryMessage},
		{"message.part.updated", CategoryMessage},
		{"message.part.removed", CategoryMessage},
		{"message.part.delta", CategoryMessage},

		// Permission
		{"permission.asked", CategoryPermission},
		{"permission.replied", CategoryPermission},

		// Question
		{"question.asked", CategoryQuestion},
		{"question.replied", CategoryQuestion},
		{"question.rejected", CategoryQuestion},

		// File (direct and remapped prefixes)
		{"file.edited", CategoryFile},
		{"file.watcher.updated", CategoryFile},
		{"vcs.branch.updated", CategoryFile},
		{"project.updated", CategoryFile},

		// Infra (direct and remapped prefixes)
		{"server.connected", CategoryInfra},
		{"server.heartbeat", CategoryInfra},
		{"server.instance.disposed", CategoryInfra},
		{"global.disposed", CategoryInfra},
		{"installation.updated", CategoryInfra},
		{"installation.update-available", CategoryInfra},
		{"lsp.updated", CategoryInfra},
		{"lsp.client.diagnostics", CategoryInfra},
		{"mcp.tools.changed", CategoryInfra},
		{"mcp.browser.open.failed", CategoryInfra},
		{"command.executed", CategoryInfra},

		// PTY
		{"pty.created", CategoryPTY},
		{"pty.updated", CategoryPTY},
		{"pty.exited", CategoryPTY},
		{"pty.deleted", CategoryPTY},

		// Workspace
		{"workspace.ready", CategoryWorkspace},
		{"workspace.failed", CategoryWorkspace},
		{"worktree.ready", CategoryWorkspace},
		{"worktree.failed", CategoryWorkspace},

		// TUI
		{"tui.prompt.append", CategoryTUI},
		{"tui.command.execute", CategoryTUI},
		{"tui.toast.show", CategoryTUI},
		{"tui.session.select", CategoryTUI},

		// Todo
		{"todo.updated", CategoryTodo},

		// Unknown types default to infra
		{"unknown.something", CategoryInfra},
		{"", CategoryInfra},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := Categorize(tt.eventType)
			if got != tt.want {
				t.Errorf("Categorize(%q) = %q, want %q", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestCategoryBadge(t *testing.T) {
	for _, cat := range AllCategories() {
		badge := CategoryBadge(cat)
		if badge == "" {
			t.Errorf("CategoryBadge(%q) returned empty string", cat)
		}
		if badge[0] != '[' || badge[len(badge)-1] != ']' {
			t.Errorf("CategoryBadge(%q) = %q, want bracketed", cat, badge)
		}
	}
}

func TestCategoryColor(t *testing.T) {
	for _, cat := range AllCategories() {
		color := CategoryColor(cat)
		if color == nil {
			t.Errorf("CategoryColor(%q) returned nil", cat)
		}
	}
}

func TestAllCategories(t *testing.T) {
	cats := AllCategories()
	if len(cats) != 10 {
		t.Errorf("AllCategories() returned %d categories, want 10", len(cats))
	}

	seen := make(map[EventCategory]bool)
	for _, c := range cats {
		if seen[c] {
			t.Errorf("duplicate category: %q", c)
		}
		seen[c] = true
	}
}

func TestCategorizeWorktree(t *testing.T) {
	// worktree.* should map to workspace since its prefix is "worktree"
	// but we need to verify it goes through the workspace path
	got := Categorize("worktree.ready")
	if got != CategoryWorkspace {
		t.Errorf("Categorize(worktree.ready) = %q, want %q", got, CategoryWorkspace)
	}
}
