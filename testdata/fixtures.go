package testdata

import (
	"time"

	"github.com/rxbynerd/monocular/internal/sse"
)

var Now = time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
var NowMillis = float64(Now.UnixMilli())

// SampleEvents contains one canonical GlobalEvent per known event type.
var SampleEvents = map[string]sse.GlobalEvent{
	// Session lifecycle
	"session.created": SessionCreated("ses_01JTEST", "fix-auth-bug", "Fix auth bug", "/Users/dev/myproject"),
	"session.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.updated",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
				"info": map[string]any{
					"id": "ses_01JTEST", "slug": "fix-auth-bug", "title": "Fix auth bug (updated)",
					"directory": "/Users/dev/myproject",
				},
			},
		},
	},
	"session.deleted": SessionDeleted("ses_01JTEST", "fix-auth-bug", "Fix auth bug", "/Users/dev/myproject"),
	"session.status":  SessionStatus("ses_01JTEST", "fix-auth-bug", "busy"),
	"session.idle":    SessionIdle("ses_01JTEST"),
	"session.compacted": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.compacted",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
			},
		},
	},
	"session.diff": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.diff",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
				"diff":      []any{"some diff content"},
			},
		},
	},
	"session.error": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.error",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
				"error":     "something went wrong",
			},
		},
	},

	// Messages
	"message.updated":      MessageUpdated("ses_01JTEST", "assistant", 0.0042, map[string]any{"input": 100, "output": 50, "reasoning": 10}),
	"message.updated.user": MessageUpdated("ses_01JTEST", "user", 0, nil),
	"message.removed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "message.removed",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
				"info":      map[string]any{"id": "msg_01JTEST"},
			},
		},
	},
	"message.part.updated": MessagePartUpdated("ses_01JTEST", "tool", map[string]any{
		"type":   "tool",
		"toolID": "bash",
		"state":  map[string]any{"status": "running"},
	}),
	"message.part.removed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "message.part.removed",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
				"part":      map[string]any{"type": "text"},
			},
		},
	},
	"message.part.delta": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "message.part.delta",
			Properties: map[string]any{
				"sessionID": "ses_01JTEST",
				"field":     "content",
				"delta":     "I'll fix the auth",
			},
		},
	},

	// Permissions
	"permission.asked":   PermissionAsked("perm_01JTEST", "ses_01JTEST", "bash", []any{"npm install"}),
	"permission.replied": PermissionReplied("perm_01JTEST", "ses_01JTEST"),

	// Questions
	"question.asked": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "question.asked",
			Properties: map[string]any{
				"id":        "q_01JTEST",
				"sessionID": "ses_01JTEST",
				"title":     "Which database?",
			},
		},
	},
	"question.replied": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "question.replied",
			Properties: map[string]any{
				"id":        "q_01JTEST",
				"sessionID": "ses_01JTEST",
			},
		},
	},
	"question.rejected": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "question.rejected",
			Properties: map[string]any{
				"id":        "q_01JTEST",
				"sessionID": "ses_01JTEST",
			},
		},
	},

	// Files & VCS
	"file.edited": FileEdited("/Users/dev/myproject", "src/auth/index.ts"),
	"file.watcher.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "file.watcher.updated",
			Properties: map[string]any{"file": "src/utils.ts"},
		},
	},
	"vcs.branch.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "vcs.branch.updated",
			Properties: map[string]any{"branch": "feature/auth-fix"},
		},
	},
	"project.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "project.updated",
			Properties: map[string]any{},
		},
	},

	// Infrastructure
	"server.connected": {
		Payload: sse.EventPayload{
			Type:       "server.connected",
			Properties: map[string]any{},
		},
	},
	"server.heartbeat": {
		Payload: sse.EventPayload{
			Type:       "server.heartbeat",
			Properties: map[string]any{},
		},
	},
	"server.instance.disposed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "server.instance.disposed",
			Properties: map[string]any{"directory": "/Users/dev/myproject"},
		},
	},
	"global.disposed": {
		Payload: sse.EventPayload{
			Type:       "global.disposed",
			Properties: map[string]any{},
		},
	},
	"installation.updated": {
		Payload: sse.EventPayload{
			Type:       "installation.updated",
			Properties: map[string]any{"version": "1.2.3"},
		},
	},
	"installation.update-available": {
		Payload: sse.EventPayload{
			Type:       "installation.update-available",
			Properties: map[string]any{"version": "1.3.0"},
		},
	},
	"lsp.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "lsp.updated",
			Properties: map[string]any{},
		},
	},
	"lsp.client.diagnostics": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "lsp.client.diagnostics",
			Properties: map[string]any{"file": "src/auth.ts", "count": float64(3)},
		},
	},
	"mcp.tools.changed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "mcp.tools.changed",
			Properties: map[string]any{},
		},
	},
	"mcp.browser.open.failed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "mcp.browser.open.failed",
			Properties: map[string]any{"url": "http://example.com"},
		},
	},
	"command.executed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "command.executed",
			Properties: map[string]any{"command": "npm test"},
		},
	},

	// PTY
	"pty.created": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "pty.created",
			Properties: map[string]any{"id": "pty_01JTEST"},
		},
	},
	"pty.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "pty.updated",
			Properties: map[string]any{"id": "pty_01JTEST"},
		},
	},
	"pty.exited": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "pty.exited",
			Properties: map[string]any{"id": "pty_01JTEST", "exitCode": float64(0)},
		},
	},
	"pty.deleted": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "pty.deleted",
			Properties: map[string]any{"id": "pty_01JTEST"},
		},
	},

	// Workspaces
	"workspace.ready": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "workspace.ready",
			Properties: map[string]any{},
		},
	},
	"workspace.failed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "workspace.failed",
			Properties: map[string]any{"error": "timeout"},
		},
	},
	"worktree.ready": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "worktree.ready",
			Properties: map[string]any{},
		},
	},
	"worktree.failed": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "worktree.failed",
			Properties: map[string]any{"error": "conflict"},
		},
	},

	// TUI
	"tui.prompt.append": {
		Payload: sse.EventPayload{
			Type:       "tui.prompt.append",
			Properties: map[string]any{"text": "fix the bug"},
		},
	},
	"tui.command.execute": {
		Payload: sse.EventPayload{
			Type:       "tui.command.execute",
			Properties: map[string]any{"command": "/clear"},
		},
	},
	"tui.toast.show": {
		Payload: sse.EventPayload{
			Type:       "tui.toast.show",
			Properties: map[string]any{"message": "Copied to clipboard"},
		},
	},
	"tui.session.select": {
		Payload: sse.EventPayload{
			Type:       "tui.session.select",
			Properties: map[string]any{"sessionID": "ses_01JTEST"},
		},
	},

	// Todos
	"todo.updated": {
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type:       "todo.updated",
			Properties: map[string]any{"id": "todo_01JTEST", "content": "Fix tests"},
		},
	},
}

// Builder functions for constructing events with custom fields.

func SessionCreated(id, slug, title, dir string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: dir,
		Payload: sse.EventPayload{
			Type: "session.created",
			Properties: map[string]any{
				"sessionID": id,
				"info": map[string]any{
					"id": id, "slug": slug, "title": title,
					"directory": dir,
					"time": map[string]any{
						"created": NowMillis,
						"updated": NowMillis,
					},
				},
			},
		},
	}
}

func SessionDeleted(id, slug, title, dir string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: dir,
		Payload: sse.EventPayload{
			Type: "session.deleted",
			Properties: map[string]any{
				"sessionID": id,
				"info": map[string]any{
					"id": id, "slug": slug, "title": title,
					"directory": dir,
				},
			},
		},
	}
}

// SessionDeletedLegacy creates a session.deleted event without the top-level
// sessionID field, requiring fallback to info.id.
func SessionDeletedLegacy(id, slug, title, dir string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: dir,
		Payload: sse.EventPayload{
			Type: "session.deleted",
			Properties: map[string]any{
				"info": map[string]any{
					"id": id, "slug": slug, "title": title,
					"directory": dir,
				},
			},
		},
	}
}

func SessionStatus(sessionID, slug, statusType string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.status",
			Properties: map[string]any{
				"sessionID": sessionID,
				"slug":      slug,
				"status":    map[string]any{"type": statusType},
			},
		},
	}
}

func SessionStatusRetry(sessionID, slug string, attempt int, message string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.status",
			Properties: map[string]any{
				"sessionID": sessionID,
				"slug":      slug,
				"status": map[string]any{
					"type":    "retry",
					"attempt": float64(attempt),
					"message": message,
				},
			},
		},
	}
}

func SessionIdle(sessionID string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "session.idle",
			Properties: map[string]any{
				"sessionID": sessionID,
			},
		},
	}
}

func MessageUpdated(sessionID, role string, cost float64, tokens map[string]any) sse.GlobalEvent {
	info := map[string]any{
		"id":   "msg_01JTEST",
		"role": role,
	}
	if cost > 0 {
		info["cost"] = cost
	}
	if tokens != nil {
		info["tokens"] = tokens
	}
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "message.updated",
			Properties: map[string]any{
				"sessionID": sessionID,
				"info":      info,
			},
		},
	}
}

func MessagePartUpdated(sessionID, partType string, part map[string]any) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "message.part.updated",
			Properties: map[string]any{
				"sessionID": sessionID,
				"part":      part,
				"time":      NowMillis,
			},
		},
	}
}

func MessagePartTool(sessionID, toolID, callID, status string) sse.GlobalEvent {
	return MessagePartUpdated(sessionID, "tool", map[string]any{
		"type":   "tool",
		"toolID": toolID,
		"id":     callID,
		"state":  map[string]any{"status": status},
	})
}

func PermissionAsked(id, sessionID, permission string, patterns []any) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "permission.asked",
			Properties: map[string]any{
				"id":         id,
				"sessionID":  sessionID,
				"permission": permission,
				"patterns":   patterns,
			},
		},
	}
}

func PermissionReplied(id, sessionID string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: "/Users/dev/myproject",
		Payload: sse.EventPayload{
			Type: "permission.replied",
			Properties: map[string]any{
				"id":        id,
				"sessionID": sessionID,
			},
		},
	}
}

func FileEdited(dir, file string) sse.GlobalEvent {
	return sse.GlobalEvent{
		Directory: dir,
		Payload: sse.EventPayload{
			Type:       "file.edited",
			Properties: map[string]any{"file": file},
		},
	}
}

// Lifecycle sequences for integration tests.

// SessionLifecycle returns events for a full session: created -> busy -> idle -> deleted.
func SessionLifecycle(id, slug, title, dir string) []sse.GlobalEvent {
	return []sse.GlobalEvent{
		SessionCreated(id, slug, title, dir),
		SessionStatus(id, slug, "busy"),
		SessionIdle(id),
		SessionDeleted(id, slug, title, dir),
	}
}

// ToolLifecycle returns events for a tool execution: pending -> running -> completed.
func ToolLifecycle(sessionID, toolID, callID string) []sse.GlobalEvent {
	return []sse.GlobalEvent{
		MessagePartTool(sessionID, toolID, callID, "pending"),
		MessagePartTool(sessionID, toolID, callID, "running"),
		MessagePartTool(sessionID, toolID, callID, "completed"),
	}
}

// PermissionLifecycle returns events for a permission request/reply cycle.
func PermissionLifecycle(id, sessionID, permission string) []sse.GlobalEvent {
	return []sse.GlobalEvent{
		PermissionAsked(id, sessionID, permission, []any{"pattern1"}),
		PermissionReplied(id, sessionID),
	}
}

// SSE wire format strings for parser tests.
var RawSSELines = map[string][]string{
	"simple_event": {
		`data: {"directory":"/Users/dev/myproject","payload":{"type":"session.status","properties":{"sessionID":"ses_01J","status":{"type":"busy"}}}}`,
		"",
	},
	"no_directory": {
		`data: {"payload":{"type":"server.connected","properties":{}}}`,
		"",
	},
	"multiline_data": {
		`data: {"directory":"/Users/dev/myproject",`,
		`data: "payload":{"type":"session.created","properties":{"sessionID":"ses_01J"}}}`,
		"",
	},
	"with_comment": {
		`: this is a comment`,
		`data: {"directory":"/d","payload":{"type":"todo.updated","properties":{}}}`,
		"",
	},
	"with_id_retry": {
		`id: 123`,
		`retry: 5000`,
		`data: {"directory":"/d","payload":{"type":"todo.updated","properties":{}}}`,
		"",
	},
	"malformed_json": {
		`data: {not valid json`,
		"",
	},
}
