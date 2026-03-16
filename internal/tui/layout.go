package tui

func (m *Model) applyFocus() {
	if m.focusLogs {
		m.processTable.Blur()
	} else {
		m.processTable.Focus()
	}
}

func (m *Model) applyLayout() {
	leftW, rightW, topH, bottomH := m.layoutDims()
	m.processTable.SetWidth(max(20, leftW-4))
	m.processTable.SetHeight(max(3, topH-3))
	m.logViewport.SetWidth(max(20, rightW-4))
	m.logViewport.SetHeight(max(3, topH-3))
	m.inspector.SetWidth(max(20, m.width-4))
	m.inspector.SetHeight(max(3, bottomH-3))
	m.help.SetWidth(max(20, m.width))
	m.applyFocus()
}

func (m Model) layoutDims() (int, int, int, int) {
	w := m.width
	if w <= 0 {
		w = 140
	}
	h := m.height
	if h <= 0 {
		h = 40
	}
	gap := 1
	leftW := ((w - gap) * m.hSplit) / 100
	rightW := w - gap - leftW
	mainAreaH := max(16, h-4)
	topH := max(10, (mainAreaH*m.vSplit)/100)
	bottomH := max(6, mainAreaH-topH)
	return leftW, rightW, topH, bottomH
}
