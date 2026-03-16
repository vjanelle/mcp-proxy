package tui

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Tab      key.Binding
	Start    key.Binding
	Stop     key.Binding
	Restart  key.Binding
	Wider    key.Binding
	Narrower key.Binding
	Taller   key.Binding
	Shorter  key.Binding
	Quit     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Start, k.Stop, k.Restart, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Start, k.Stop, k.Restart, k.Quit},
		{k.Narrower, k.Wider, k.Shorter, k.Taller},
		{
			key.NewBinding(key.WithKeys("j/k"), key.WithHelp("j/k", "log line nav")),
			key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "follow tail")),
		},
	}
}

func newKeyMap() keyMap {
	return keyMap{
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
		Start:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
		Stop:     key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop")),
		Restart:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
		Narrower: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "narrower left")),
		Wider:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "wider left")),
		Shorter:  key.NewBinding(key.WithKeys("-", "_"), key.WithHelp("-", "shorter top")),
		Taller:   key.NewBinding(key.WithKeys("=", "+"), key.WithHelp("=", "taller top")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
