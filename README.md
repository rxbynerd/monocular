# Monocular

A read-only TUI dashboard for observing [OpenCode](https://opencode.ai) server SSE event streams in real time.

Monocular connects to a running OpenCode server's `/global/event` endpoint and displays live session activity, tool executions, cost tracking, permission requests, and a full event log. It is a diagnostic/observability tool -- it never sends prompts or modifies server state.

<!-- TODO: Add screenshot/recording of the TUI in action -->

## Features

- **Three-column TUI layout** -- sessions list, scrollable event log, and event detail panel
- **Live session tracking** -- status, model, cost, and token usage per session
- **Tool execution tracking** -- see active tool calls with elapsed time
- **Permission and question alerts** -- highlighted notifications for events requiring attention
- **Event category filtering** -- toggle visibility of session, message, permission, file, infra, and other event types
- **NDJSON output mode** -- `--json` flag for piping structured event data to other tools
- **Automatic reconnection** -- exponential backoff with state recovery on reconnect
- **Cross-platform** -- single static binary for macOS (arm64/amd64), Linux (amd64/arm64), and Windows (amd64)

## Requirements

- A running [OpenCode](https://opencode.ai) server (Monocular connects to its SSE endpoint)
- Go 1.26+ (only if building from source)

## Installation

### From release binaries

Download the latest binary for your platform from the [Releases](https://github.com/rxbynerd/monocular/releases) page. Checksums are provided for verification.

```sh
# Example: macOS ARM
curl -LO https://github.com/rxbynerd/monocular/releases/latest/download/monocular-darwin-arm64
chmod +x monocular-darwin-arm64
mv monocular-darwin-arm64 /usr/local/bin/monocular
```

### From source

```sh
git clone https://github.com/rxbynerd/monocular.git
cd monocular
make build
# Binary is written to ./monocular
```

## Usage

```sh
# Connect to the default OpenCode server at http://127.0.0.1:4096
monocular

# Connect to a specific server URL
monocular -u http://localhost:8080

# Filter to only show session and permission events
monocular -f session,permission

# Output raw events as NDJSON (no TUI)
monocular --json

# Combine flags
monocular -u http://10.0.0.5:4096 -f session,message --json
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--url` | `-u` | `http://127.0.0.1:4096` | OpenCode server URL |
| `--directory` | `-d` | | Initial client-side directory filter |
| `--filter` | `-f` | all | Comma-separated event categories to show |
| `--json` | | `false` | Output raw events as NDJSON to stdout (no TUI) |
| `--no-color` | | `false` | Disable colors |
| `--version` | `-v` | | Print version |

### Event categories for `--filter`

`session`, `message`, `permission`, `question`, `file`, `infra`, `pty`, `workspace`, `tui`, `todo`

### TUI keyboard shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate event log |
| `Tab` | Cycle focus between panels |
| `f` | Open category filter picker |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

## Known limitations

- **Read-only** -- Monocular cannot send prompts or interact with OpenCode sessions; it is strictly an observer.
- **No event persistence** -- events are held in memory only for the current session; closing Monocular discards the event history.
- **Single server** -- connects to one OpenCode instance at a time.
- **No `/global/sync-event` support** -- only the `/global/event` SSE stream is consumed.
- **Terminal size** -- the three-column layout requires a reasonably wide terminal (80+ columns recommended); narrow terminals will clip content.

## Architecture

Monocular is a [Bubble Tea v2](https://charm.land/bubbletea/v2) application following the Elm architecture (Model-Update-View). See [`CLAUDE.md`](CLAUDE.md) for detailed architecture notes and package responsibilities.

**Data flow:**

```
SSE stream -> sse.Connect() -> channels -> Bubble Tea Cmd/Msg loop
  -> model.ApplyEvent() mutates state -> ui.App.View() renders
```

## Contributing

Contributions are welcome. Please open an issue to discuss significant changes before submitting a pull request.

```sh
make test       # run tests
make test-race  # run tests with race detector
make lint       # run go vet
```

## License

Apache License 2.0. See [LICENSE](LICENSE) for the full text.
