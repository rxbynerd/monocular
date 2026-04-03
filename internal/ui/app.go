package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/rxbynerd/monocular/internal/model"
	"github.com/rxbynerd/monocular/internal/sse"
)

const streamingTimeout = 3 * time.Second

// AppConfig holds configuration for the TUI app.
type AppConfig struct {
	URL       string
	Directory string
	Filter    map[sse.EventCategory]bool
	NoColor   bool
}

// App is the root Bubble Tea model.
type App struct {
	cfg   AppConfig
	state *model.DashboardState

	// Sub-components
	connBar      ConnectionBar
	sessionPanel SessionPanel
	toolTracker  ToolTracker
	eventLog     EventLog
	detailPanel  DetailPanel
	alertBar     AlertBar
	helpBar      HelpBar
	filterPicker FilterPicker
	searchInput  textinput.Model

	// Layout
	width  int
	height int

	// Connection
	cancel context.CancelFunc
}

// NewApp creates a new App model.
func NewApp(cfg AppConfig) App {
	state := model.NewDashboardState()
	state.Connection.URL = cfg.URL
	if cfg.Filter != nil {
		state.UI.Filter = cfg.Filter
	}
	if cfg.Directory != "" {
		state.UI.Directory = cfg.Directory
	}

	si := textinput.New()
	si.Placeholder = "Search events..."

	return App{
		cfg:          cfg,
		state:        state,
		connBar:      NewConnectionBar(),
		sessionPanel: NewSessionPanel(),
		toolTracker:  NewToolTracker(),
		eventLog:     NewEventLog(),
		detailPanel:  NewDetailPanel(),
		alertBar:     NewAlertBar(),
		helpBar:      NewHelpBar(),
		filterPicker: NewFilterPicker(),
		searchInput:  si,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.connectSSE(),
		tickCmd(),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLayout()
		return a, nil

	case sseEventWithContinue:
		now := time.Now()
		model.ApplyEvent(a.state, msg.event, now)
		return a, waitForSSEEvent(msg.ctx, msg.ch)

	case sseStateWithContinue:
		a.state.Connection.State = msg.state
		cmd := waitForSSEState(msg.ctx, msg.ch)
		if msg.state == sse.Connected && a.state.Connection.ReconnectCount > 0 {
			return a, tea.Batch(cmd, a.fetchSessionsCmd())
		}
		return a, cmd

	case model.SessionsRefreshedMsg:
		model.ApplySessionsRefreshed(a.state, msg.Sessions, time.Now())
		return a, nil

	case model.TickMsg:
		now := time.Now()
		model.ClearStreamingIndicators(a.state, now, streamingTimeout)
		return a, tickCmd()

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a App) View() tea.View {
	if a.width == 0 || a.height == 0 {
		v := tea.NewView("Initializing...")
		v.AltScreen = true
		return v
	}

	now := time.Now()

	// Filter picker overlay
	if a.state.UI.ShowFilter {
		v := tea.NewView(a.filterPicker.View(a.state, a.width, a.height))
		v.AltScreen = true
		return v
	}

	// Connection bar (top)
	topBar := a.connBar.View(a.state, now)

	// Three columns
	leftWidth := a.width * 30 / 100
	centerWidth := a.width * 40 / 100
	rightWidth := a.width - leftWidth - centerWidth

	panelHeight := a.height - 4

	// Left: sessions + tools
	sessionsView := PanelStyle(a.state.UI.FocusedPanel == model.PanelSessions, leftWidth-2, panelHeight-2).
		Render(styleTitle.Render(" Sessions") + "\n" + a.sessionPanel.View(a.state) + "\n" + a.toolTracker.View(a.state, now))

	// Center: event log
	searchLine := ""
	if a.state.UI.SearchActive {
		searchLine = " " + a.searchInput.View() + "\n"
	}
	pauseIndicator := ""
	if a.state.UI.Paused {
		pauseIndicator = styleAlertError.Render(" PAUSED") + "\n"
	}
	eventsView := PanelStyle(a.state.UI.FocusedPanel == model.PanelEvents, centerWidth-2, panelHeight-2).
		Render(styleTitle.Render(" Events") + pauseIndicator + searchLine + "\n" + a.eventLog.View(a.state))

	// Right: detail
	detailView := PanelStyle(a.state.UI.FocusedPanel == model.PanelDetail, rightWidth-2, panelHeight-2).
		Render(styleTitle.Render(" Detail") + "\n" + a.detailPanel.View(a.state))

	columns := lipgloss.JoinHorizontal(lipgloss.Top, sessionsView, eventsView, detailView)

	// Alert bar
	alertView := a.alertBar.View(a.state, now)

	// Help bar (bottom)
	helpView := a.helpBar.View()

	// Compose
	parts := []string{topBar, columns}
	if alertView != "" {
		parts = append(parts, alertView)
	}
	parts = append(parts, helpView)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	v.AltScreen = true
	return v
}

func (a App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Search mode captures all keys
	if a.state.UI.SearchActive {
		switch key {
		case "escape":
			a.state.UI.SearchActive = false
			a.state.UI.SearchQuery = ""
			a.searchInput.Reset()
			return a, nil
		case "enter":
			a.state.UI.SearchQuery = a.searchInput.Value()
			a.state.UI.SearchActive = false
			return a, nil
		default:
			var cmd tea.Cmd
			a.searchInput, cmd = a.searchInput.Update(msg)
			a.state.UI.SearchQuery = a.searchInput.Value()
			return a, cmd
		}
	}

	// Filter picker mode
	if a.state.UI.ShowFilter {
		switch key {
		case "f", "escape":
			a.state.UI.ShowFilter = false
		case "j", "down":
			a.filterPicker.MoveDown()
		case "k", "up":
			a.filterPicker.MoveUp()
		case " ":
			a.filterPicker.Toggle(a.state)
		}
		return a, nil
	}

	// Global keys
	switch key {
	case "q", "ctrl+c":
		if a.cancel != nil {
			a.cancel()
		}
		return a, tea.Quit

	case "tab":
		a.cycleFocus(1)
	case "shift+tab":
		a.cycleFocus(-1)
	case "1":
		a.state.UI.FocusedPanel = model.PanelSessions
	case "2":
		a.state.UI.FocusedPanel = model.PanelEvents
	case "3":
		a.state.UI.FocusedPanel = model.PanelDetail

	case "j", "down":
		a.scrollFocused(1)
	case "k", "up":
		a.scrollFocused(-1)
	case "g", "home":
		a.gotoTopFocused()
	case "G", "end":
		a.gotoBottomFocused()

	case "enter":
		a.handleEnter()

	case "escape":
		a.state.UI.SelectedSessionID = ""
		a.state.UI.SearchQuery = ""

	case "f":
		a.state.UI.ShowFilter = true

	case "/":
		a.state.UI.SearchActive = true
		cmd := a.searchInput.Focus()
		return a, cmd

	case "p":
		if a.state.UI.Paused {
			a.state.UI.Paused = false
			model.FlushPauseBuffer(a.state)
		} else {
			a.state.UI.Paused = true
		}

	case "c":
		a.state.Events = nil
		a.state.UI.SelectedEventIdx = -1

	case "?":
		a.state.UI.ShowHelp = !a.state.UI.ShowHelp
	}

	return a, nil
}

func (a *App) cycleFocus(dir int) {
	panels := []model.Panel{model.PanelSessions, model.PanelEvents, model.PanelDetail}
	current := int(a.state.UI.FocusedPanel)
	next := (current + dir + len(panels)) % len(panels)
	a.state.UI.FocusedPanel = panels[next]
}

func (a *App) scrollFocused(n int) {
	switch a.state.UI.FocusedPanel {
	case model.PanelSessions:
		if n > 0 {
			a.sessionPanel.ScrollDown(n)
		} else {
			a.sessionPanel.ScrollUp(-n)
		}
	case model.PanelEvents:
		if n > 0 {
			a.eventLog.ScrollDown(n)
		} else {
			a.eventLog.ScrollUp(-n)
		}
	case model.PanelDetail:
		if n > 0 {
			a.detailPanel.ScrollDown(n)
		} else {
			a.detailPanel.ScrollUp(-n)
		}
	}
}

func (a *App) gotoTopFocused() {
	switch a.state.UI.FocusedPanel {
	case model.PanelSessions:
		a.sessionPanel.GotoTop()
	case model.PanelEvents:
		a.eventLog.GotoTop()
	case model.PanelDetail:
		a.detailPanel.GotoTop()
	}
}

func (a *App) gotoBottomFocused() {
	switch a.state.UI.FocusedPanel {
	case model.PanelSessions:
		a.sessionPanel.GotoBottom(len(a.state.SessionOrder))
	case model.PanelEvents:
		a.eventLog.GotoBottom()
	case model.PanelDetail:
		a.detailPanel.GotoBottom()
	}
}

func (a *App) handleEnter() {
	switch a.state.UI.FocusedPanel {
	case model.PanelSessions:
		sid := a.sessionPanel.SelectedSessionID(a.state)
		if a.state.UI.SelectedSessionID == sid {
			a.state.UI.SelectedSessionID = ""
		} else {
			a.state.UI.SelectedSessionID = sid
		}
	case model.PanelEvents:
		idx := a.eventLog.SelectedIdx()
		if idx >= 0 && idx < len(a.state.Events) {
			a.state.UI.SelectedEventIdx = idx
			a.state.UI.FocusedPanel = model.PanelDetail
		}
	}
}

func (a *App) updateLayout() {
	a.connBar.SetWidth(a.width)
	a.helpBar.SetWidth(a.width)
	a.alertBar.SetWidth(a.width)

	leftWidth := a.width * 30 / 100
	centerWidth := a.width * 40 / 100
	rightWidth := a.width - leftWidth - centerWidth

	panelHeight := a.height - 6

	a.sessionPanel.SetSize(leftWidth-4, panelHeight/2)
	a.toolTracker.SetWidth(leftWidth - 4)
	a.eventLog.SetSize(centerWidth-4, panelHeight)
	a.detailPanel.SetSize(rightWidth-4, panelHeight)
}

// connectSSE starts the SSE connection and returns commands that feed events
// into the Bubble Tea update loop.
func (a *App) connectSSE() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	eventCh := make(chan sse.GlobalEvent, 256)
	stateCh := make(chan sse.ConnectionState, 10)

	sseConfig := sse.DefaultConfig(a.cfg.URL)
	sseConfig.Directory = a.cfg.Directory

	go sse.Connect(ctx, sseConfig, eventCh, stateCh)

	return tea.Batch(
		waitForSSEEvent(ctx, eventCh),
		waitForSSEState(ctx, stateCh),
	)
}

func waitForSSEEvent(ctx context.Context, ch <-chan sse.GlobalEvent) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-ch:
			return sseEventWithContinue{event: ev, ctx: ctx, ch: ch}
		}
	}
}

func waitForSSEState(ctx context.Context, ch <-chan sse.ConnectionState) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case state := <-ch:
			return sseStateWithContinue{state: state, ctx: ctx, ch: ch}
		}
	}
}

type sseEventWithContinue struct {
	event sse.GlobalEvent
	ctx   context.Context
	ch    <-chan sse.GlobalEvent
}

type sseStateWithContinue struct {
	state sse.ConnectionState
	ctx   context.Context
	ch    <-chan sse.ConnectionState
}

func (a *App) fetchSessionsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sessions, err := sse.FetchSessions(ctx, a.cfg.URL)
		if err != nil {
			return nil
		}
		return model.SessionsRefreshedMsg{Sessions: sessions}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return model.TickMsg{}
	})
}
