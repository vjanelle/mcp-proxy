package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func runIndicator(running bool) string {
	if running {
		return "RUN"
	}
	return "STOP"
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// SetLogRedactor installs an optional render-time redactor for TUI log lines.
// It only affects what is displayed in the TUI; underlying events remain raw.
func (m *Model) SetLogRedactor(redactor func(string) string) {
	if redactor == nil {
		m.redactLog = noopRedactor
		return
	}

	m.redactLog = redactor
}
