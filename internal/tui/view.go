package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	if len(m.statuses) == 0 {
		view := tea.NewView("No configured processes.\n")
		view.AltScreen = true
		view.MouseMode = tea.MouseModeCellMotion
		return view
	}

	leftW, rightW, topH, bottomH := m.layoutDims()

	leftTitle := titleStyle.Render("Processes")
	rightTitle := titleStyle.Render("Debug Log")
	if m.focusLogs {
		rightTitle = focusStyle.Render("Debug Log (focused)")
	} else {
		leftTitle = focusStyle.Render("Processes (focused)")
	}

	leftPanel := panelStyle.Width(leftW).Height(topH).Render(leftTitle + "\n" + m.processTable.View())
	rightPanel := panelStyle.Width(rightW).Height(topH).Render(rightTitle + "\n" + m.logViewport.View())
	top := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)

	selectedPort := "-"
	if m.statuses[m.selected].Port > 0 {
		selectedPort = fmt.Sprintf("%d", m.statuses[m.selected].Port)
	}
	selHeader := titleStyle.Render(
		fmt.Sprintf("Selected: %s on http://127.0.0.1:%s/rpc", m.statuses[m.selected].Name, selectedPort),
	)

	inspectorPanel := panelStyle.Width(m.width).Height(bottomH).Render(
		titleStyle.Render("Selected Log Line") + "\n" + m.inspector.View(),
	)
	helpLine := footerStyle.Render(m.help.View(m.keys))

	content := lipgloss.JoinVertical(lipgloss.Left, top, selHeader, inspectorPanel, helpLine)
	view := tea.NewView(content + "\n")
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}
