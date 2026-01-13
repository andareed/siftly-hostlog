package main

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) runCommand() tea.Cmd {
	switch m.ui.command.cmd {
	case CmdJump:
		if n, err := strconv.Atoi(m.ui.command.buf); err == nil {
			return m.jumpToLine(n)
		}
		return m.startNotice("Invalid line number", "warn", noticeDuration)

	case CmdSearch:
		m.setSearchQuery(m.ui.command.buf)
		if m.searchNext() {
			return nil
		}
		return m.startNotice("No matches", "warn", noticeDuration)

	case CmdFilter:
		m.setFilterPattern(m.ui.command.buf)
		return nil

	case CmdComment:
		m.addComment(m.ui.command.buf)
		return m.startNotice("Comment added", "", noticeDuration)
	}
	return nil
}

func (m *model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// universal cancel
	if msg.Type == tea.KeyEsc {
		cmd := m.exitCommand(true)
		return m, cmd
	}

	// constrained command: mark
	if m.ui.command.cmd == CmdMark {
		return m.handleMarkCommandKey(msg) // your tightened function
	}

	// commit
	if msg.Type == tea.KeyEnter {
		cmd := m.runCommand() // returns tea.Cmd or nil
		exitCmd := m.exitCommand(true)
		return m, tea.Batch(cmd, exitCmd)
	}

	// editing
	switch msg.Type {
	case tea.KeyBackspace:
		if len(m.ui.command.buf) > 0 {
			m.ui.command.buf = m.ui.command.buf[:len(m.ui.command.buf)-1]
		}
		return m, nil
	}

	// append printable rune
	if len(msg.Runes) == 1 {
		m.ui.command.buf += string(msg.Runes[0])
	}
	return m, nil
}
