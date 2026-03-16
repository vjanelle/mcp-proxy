package tui

import "charm.land/lipgloss/v2"

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	focusStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	footerStyle = lipgloss.NewStyle().Faint(true)
)
