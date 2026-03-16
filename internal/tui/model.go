package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/vjanelle/mcp-proxy/internal/proxy"
)

type tickMsg struct{}

// Model is the Bubble Tea model for the interactive proxy console.
type Model struct {
	manager  *proxy.Manager
	statuses []proxy.ProcessStatus
	selected int

	focusLogs  bool
	logCursor  int
	logLines   []string
	followLogs bool

	hSplit int
	vSplit int

	width  int
	height int

	processTable table.Model
	logViewport  viewport.Model
	inspector    viewport.Model
	help         help.Model
	keys         keyMap
	redactLog    logRedactor
}

// New constructs a ready-to-run TUI model bound to a process manager.
func New(manager *proxy.Manager) Model {
	cols := []table.Column{
		{Title: "state", Width: 6},
		{Title: "name", Width: 16},
		{Title: "port", Width: 6},
		{Title: "pid", Width: 7},
		{Title: "req", Width: 6},
		{Title: "rsp", Width: 6},
		{Title: "restart", Width: 8},
		{Title: "command", Width: 40},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(80),
	)

	lv := viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))
	lv.MouseWheelEnabled = true

	ins := viewport.New(viewport.WithWidth(80), viewport.WithHeight(6))
	ins.SoftWrap = true

	return Model{
		manager:      manager,
		hSplit:       50,
		vSplit:       62,
		followLogs:   true,
		processTable: t,
		logViewport:  lv,
		inspector:    ins,
		help:         help.New(),
		keys:         newKeyMap(),
		redactLog:    noopRedactor,
	}
}

// Init implements tea.Model and starts periodic refresh ticks.
func (m Model) Init() tea.Cmd {
	return tick()
}
