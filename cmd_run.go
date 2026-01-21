package main

import (
	"strconv"
	"unicode"

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
	case tea.KeyBackspace, tea.KeyCtrlH, tea.KeyDelete:
		m.ui.command.buf = deleteLastRune(m.ui.command.buf)
		return m, nil
	case tea.KeyCtrlW:
		m.ui.command.buf = deletePrevWord(m.ui.command.buf)
		return m, nil
	}

	// append printable rune(s), including paste blocks
	if len(msg.Runes) > 0 {
		m.ui.command.buf += string(msg.Runes)
	}
	return m, nil
}

func deleteLastRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

func deletePrevWord(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	i := len(r)
	for i > 0 && unicode.IsSpace(r[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(r[i-1]) {
		i--
	}
	return string(r[:i])
}
