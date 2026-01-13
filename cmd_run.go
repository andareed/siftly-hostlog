package main

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) runCommand() tea.Cmd {
	switch m.ci.cmd {
	case CmdJump:
		if n, err := strconv.Atoi(m.ci.buf); err == nil {
			return m.jumpToLine(n)
		}
		return m.startNotice("Invalid line number", "warn", noticeDuration)

	case CmdSearch:
		m.searchOnce(m.ci.buf)
		return nil

	case CmdFilter:
		m.setFilterPattern(m.ci.buf)
		return nil

	case CmdComment:
		m.addComment(m.ci.buf)
		m.viewport.SetContent(m.renderTable())
		return m.startNotice("Comment added", "", noticeDuration)
	}
	return nil
}

func (m *model) exitCommandMode() {
	m.ci = CommandInput{}
	m.currentMode = modeView
}

func (m *model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// universal cancel
	if msg.Type == tea.KeyEsc {
		m.exitCommandMode()
		return m, nil
	}

	// constrained command: mark
	if m.ci.cmd == CmdMark {
		return m.handleMarkCommandKey(msg) // your tightened function
	}

	// commit
	if msg.Type == tea.KeyEnter {
		cmd := m.runCommand() // returns tea.Cmd or nil
		m.exitCommandMode()
		m.viewport.SetContent(m.renderTable())
		return m, cmd
	}

	// editing
	switch msg.Type {
	case tea.KeyBackspace:
		if len(m.ci.buf) > 0 {
			m.ci.buf = m.ci.buf[:len(m.ci.buf)-1]
		}
		return m, nil
	}

	// append printable rune
	if len(msg.Runes) == 1 {
		m.ci.buf += string(msg.Runes[0])
	}
	return m, nil
}
