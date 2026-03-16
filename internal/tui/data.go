package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) syncViews() {
	rows := make([]table.Row, 0, len(m.statuses))
	for _, status := range m.statuses {
		port := "-"
		if status.Port > 0 {
			port = fmt.Sprintf("%d", status.Port)
		}
		rows = append(rows, table.Row{
			runIndicator(status.Running),
			status.Name,
			port,
			fmt.Sprintf("%d", status.PID),
			fmt.Sprintf("%d", status.Requests),
			fmt.Sprintf("%d", status.Responses),
			fmt.Sprintf("%d", status.Restarts),
			status.Command,
		})
	}
	m.processTable.SetRows(rows)
	m.processTable.SetCursor(m.selected)

	m.logLines = m.currentLogLines()
	if len(m.logLines) == 0 {
		m.logLines = []string{"(no logs yet)"}
	}
	if m.followLogs {
		m.logCursor = len(m.logLines) - 1
	}
	if m.logCursor >= len(m.logLines) {
		m.logCursor = len(m.logLines) - 1
	}
	if m.logCursor < 0 {
		m.logCursor = 0
	}

	lines := make([]string, 0, len(m.logLines))
	for i, line := range m.logLines {
		prefix := "  "
		if i == m.logCursor {
			prefix = "> "
		}
		lines = append(lines, prefix+line)
	}
	m.logViewport.SetContent(strings.Join(lines, "\n"))
	m.ensureLogCursorVisible()

	selected := ""
	if m.logCursor < len(m.logLines) {
		selected = m.logLines[m.logCursor]
	}
	m.inspector.SetContent(selected)
}

func (m *Model) ensureLogCursorVisible() {
	h := m.logViewport.Height()
	if h <= 0 {
		return
	}
	y := m.logViewport.YOffset()
	if m.logCursor < y {
		m.logViewport.SetYOffset(m.logCursor)
		return
	}
	if m.logCursor >= y+h {
		m.logViewport.SetYOffset(m.logCursor - h + 1)
	}
}

func (m Model) currentLogLines() []string {
	if len(m.statuses) == 0 {
		return nil
	}
	selectedName := m.statuses[m.selected].Name
	events := m.manager.EventSnapshot(selectedName, 80)
	lines := make([]string, 0, len(events)*2)
	for _, evt := range events {
		header := fmt.Sprintf("%s [%s] %s", evt.Time.Format("15:04:05"), evt.Process, evt.Type)
		lines = append(lines, sanitizeLogLine(header, m.redactLog))
		lines = append(lines, "  "+sanitizeLogLine(evt.Message, m.redactLog))
	}
	return lines
}

func (m Model) handleMouse(mouse tea.Mouse) Model {
	leftW, _, topH, _ := m.layoutDims()
	if mouse.Y < 0 || mouse.X < 0 {
		return m
	}

	if mouse.Y <= topH {
		if mouse.X < leftW {
			m.focusLogs = false
			m.applyFocus()
			row := mouse.Y - 3
			if row >= 0 {
				idx := row
				if idx >= 0 && idx < len(m.statuses) {
					m.selected = idx
					m.processTable.SetCursor(idx)
					m.syncViews()
				}
			}
			return m
		}

		m.focusLogs = true
		m.applyFocus()
		if mouse.Button == tea.MouseWheelUp && m.logCursor > 0 {
			m.logCursor--
			m.followLogs = false
			m.syncViews()
		}
		if mouse.Button == tea.MouseWheelDown {
			if m.logCursor < len(m.logLines)-1 {
				m.logCursor++
				m.followLogs = m.logCursor >= len(m.logLines)-1
				m.syncViews()
			}
		}
	}

	return m
}
