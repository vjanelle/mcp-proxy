package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vjanelle/mcp-proxy/internal/config"
	"github.com/vjanelle/mcp-proxy/internal/proxy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Model", func() {
	It("renders and responds to resize/focus/split controls", func() {
		manager := proxy.NewManager([]config.ProcessConfig{
			{Name: "fake", Command: "echo", Port: 18081},
		})
		m := New(manager)

		model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		m = model.(Model)
		model, _ = m.Update(tickMsg{})
		m = model.(Model)
		view := m.View()
		Expect(view.Content).To(ContainSubstring("Processes"))

		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		m = model.(Model)
		Expect(m.focusLogs).To(BeTrue())

		beforeHSplit := m.hSplit
		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "]"}))
		m = model.(Model)
		Expect(m.hSplit).To(BeNumerically(">=", beforeHSplit))

		beforeVSplit := m.vSplit
		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "-"}))
		m = model.(Model)
		Expect(m.vSplit).To(BeNumerically("<=", beforeVSplit))
	})

	It("moves log cursor and follow mode in log focus", func() {
		manager := proxy.NewManager([]config.ProcessConfig{
			{Name: "fake", Command: "echo", Port: 18081},
		})
		m := New(manager)
		m.statuses = manager.List()
		m.focusLogs = true
		m.syncViews()

		model, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		m = model.(Model)
		Expect(m.logCursor).To(BeNumerically(">=", 0))

		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
		m = model.(Model)
		Expect(m.followLogs).To(BeFalse())

		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
		m = model.(Model)
		Expect(m.followLogs).To(BeTrue())
		Expect(m.logCursor).To(Equal(len(m.logLines) - 1))
	})

	It("renders empty state and handles mouse focus changes", func() {
		m := New(proxy.NewManager(nil))
		v := m.View()
		Expect(v.Content).To(ContainSubstring("No configured processes"))

		manager := proxy.NewManager([]config.ProcessConfig{
			{Name: "fake", Command: "echo", Port: 18081},
		})
		m = New(manager)
		model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		m = model.(Model)
		model, _ = m.Update(tickMsg{})
		m = model.(Model)

		m = m.handleMouse(tea.Mouse{X: 1, Y: 4, Button: tea.MouseLeft})
		Expect(m.focusLogs).To(BeFalse())

		m = m.handleMouse(tea.Mouse{X: 80, Y: 4, Button: tea.MouseWheelDown})
		Expect(m.focusLogs).To(BeTrue())
	})

	It("handles process control keys and unknown messages", func() {
		manager := proxy.NewManager([]config.ProcessConfig{
			{Name: "fake", Command: "echo", Port: 18081},
		})
		m := New(manager)
		model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		m = model.(Model)
		model, _ = m.Update(tickMsg{})
		m = model.(Model)

		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "s"}))
		m = model.(Model)
		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "x"}))
		m = model.(Model)
		model, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "r"}))
		m = model.(Model)

		model, _ = m.Update(struct{ tea.Msg }{})
		m = model.(Model)
		Expect(m.View().Content).To(ContainSubstring("Selected Log Line"))
	})

	It("escapes terminal control sequences for safe rendering", func() {
		input := "ok\x1b[31mred\x9b2J\x07done"
		escaped := escapeForTerminal(input)
		Expect(escaped).To(ContainSubstring("\\x1b"))
		Expect(escaped).To(ContainSubstring("\\x9b"))
		Expect(escaped).To(ContainSubstring("\\u0007"))
		Expect(strings.ContainsRune(escaped, 0x1b)).To(BeFalse())
		Expect(strings.ContainsRune(escaped, 0x9b)).To(BeFalse())
	})

	It("keeps printable log text unchanged", func() {
		plain := "request id=42 method=tools/list"
		Expect(escapeForTerminal(plain)).To(Equal(plain))
	})
})
