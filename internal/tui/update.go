package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyLayout()
		m.syncViews()
		return m, nil
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		m = m.handleMouse(msg.Mouse())
		return m, nil
	case tickMsg:
		m.statuses = m.manager.List()
		if len(m.statuses) > 0 && m.selected >= len(m.statuses) {
			m.selected = len(m.statuses) - 1
		}
		m.syncViews()
		return m, tick()
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Tab) {
		m.focusLogs = !m.focusLogs
		m.applyFocus()
		return m, nil
	}
	if key.Matches(msg, m.keys.Narrower) && m.hSplit > 25 {
		m.hSplit -= 5
		m.applyLayout()
		m.syncViews()
		return m, nil
	}
	if key.Matches(msg, m.keys.Wider) && m.hSplit < 75 {
		m.hSplit += 5
		m.applyLayout()
		m.syncViews()
		return m, nil
	}
	if key.Matches(msg, m.keys.Shorter) && m.vSplit > 45 {
		m.vSplit -= 5
		m.applyLayout()
		m.syncViews()
		return m, nil
	}
	if key.Matches(msg, m.keys.Taller) && m.vSplit < 80 {
		m.vSplit += 5
		m.applyLayout()
		m.syncViews()
		return m, nil
	}
	if key.Matches(msg, m.keys.Start) && !m.focusLogs && len(m.statuses) > 0 {
		_ = m.manager.Start(m.statuses[m.selected].Name)
		return m, nil
	}
	if key.Matches(msg, m.keys.Stop) && !m.focusLogs && len(m.statuses) > 0 {
		_ = m.manager.Stop(m.statuses[m.selected].Name)
		return m, nil
	}
	if key.Matches(msg, m.keys.Restart) && !m.focusLogs && len(m.statuses) > 0 {
		_ = m.manager.Restart(m.statuses[m.selected].Name)
		return m, nil
	}

	if m.focusLogs {
		switch msg.String() {
		case "up", "k":
			if m.logCursor > 0 {
				m.logCursor--
			}
			m.followLogs = false
			m.syncViews()
			return m, nil
		case "down", "j":
			if m.logCursor < len(m.logLines)-1 {
				m.logCursor++
			}
			m.followLogs = m.logCursor >= len(m.logLines)-1
			m.syncViews()
			return m, nil
		case "g", "home":
			m.logCursor = 0
			m.followLogs = false
			m.syncViews()
			return m, nil
		case "G", "end":
			m.logCursor = max(0, len(m.logLines)-1)
			m.followLogs = true
			m.syncViews()
			return m, nil
		default:
			var cmd tea.Cmd
			m.logViewport, cmd = m.logViewport.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.processTable, cmd = m.processTable.Update(msg)
	m.selected = m.processTable.Cursor()
	m.syncViews()
	return m, cmd
}
