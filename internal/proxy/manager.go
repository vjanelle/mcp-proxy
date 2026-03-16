package proxy

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/vjanelle/mcp-proxy/internal/config"
)

const maxEvents = 2000

// Manager coordinates process lifecycle and RPC routing for configured MCP
// subprocesses.
type Manager struct {
	mu        sync.RWMutex
	processes map[string]*managedProcess
	events    []Event
}

// NewManager constructs a manager for the provided process configs.
func NewManager(processes []config.ProcessConfig) *Manager {
	manager := &Manager{
		processes: map[string]*managedProcess{},
		events:    make([]Event, 0, maxEvents),
	}

	for _, processCfg := range processes {
		cfg := processCfg
		manager.processes[cfg.Name] = newManagedProcess(cfg, manager.addEvent)
	}

	return manager
}

// StartAutoProcesses starts all processes flagged with AutoStart.
func (m *Manager) StartAutoProcesses() error {
	for _, name := range m.Names() {
		process := m.get(name)
		if process == nil {
			continue
		}

		if !process.cfg.AutoStart {
			continue
		}

		if err := process.start(); err != nil {
			return fmt.Errorf("start process %q: %w", name, err)
		}
	}

	return nil
}

// Start starts a named process.
func (m *Manager) Start(name string) error {
	process, err := m.process(name)
	if err != nil {
		return err
	}

	return process.start()
}

// Stop stops a named process.
func (m *Manager) Stop(name string) error {
	process, err := m.process(name)
	if err != nil {
		return err
	}

	return process.stop()
}

// Restart restarts a named process.
func (m *Manager) Restart(name string) error {
	process, err := m.process(name)
	if err != nil {
		return err
	}

	return process.restart()
}

// StopAll attempts to stop every configured process.
func (m *Manager) StopAll() {
	for _, name := range m.Names() {
		_ = m.Stop(name)
	}
}

// DoRPC forwards a JSON-RPC payload to the named process and waits for a
// response when applicable.
func (m *Manager) DoRPC(ctx context.Context, name string, payload []byte) ([]byte, error) {
	process, err := m.process(name)
	if err != nil {
		return nil, err
	}

	return process.doRPC(ctx, payload)
}

// List returns a sorted snapshot of process statuses.
func (m *Manager) List() []ProcessStatus {
	names := m.Names()
	statuses := make([]ProcessStatus, 0, len(names))

	for _, name := range names {
		process := m.get(name)
		if process == nil {
			continue
		}

		statuses = append(statuses, process.status())
	}

	return statuses
}

// Names returns all managed process names in lexical order.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.processes))
	for name := range m.processes {
		names = append(names, name)
	}

	slices.Sort(names)
	return names
}

// EventSnapshot returns up to limit most-recent events, optionally filtered by
// processName when non-empty.
func (m *Manager) EventSnapshot(processName string, limit int) []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filtered := make([]Event, 0, limit)
	for i := len(m.events) - 1; i >= 0; i-- {
		evt := m.events[i]
		if processName != "" && evt.Process != processName {
			continue
		}

		filtered = append(filtered, evt)
		if len(filtered) >= limit {
			break
		}
	}

	slices.Reverse(filtered)
	return filtered
}

func (m *Manager) process(name string) (*managedProcess, error) {
	process := m.get(name)
	if process == nil {
		return nil, fmt.Errorf("unknown process %q", name)
	}

	return process, nil
}

func (m *Manager) get(name string) *managedProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processes[name]
}

func (m *Manager) addEvent(evt Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.events) == maxEvents {
		copy(m.events, m.events[1:])
		m.events = m.events[:maxEvents-1]
	}

	m.events = append(m.events, evt)
}
