package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Command int

const (
	CmdNone Command = iota
	CmdJump
	CmdSearch
	CmdFilter
	CmdComment
	CmdMark
)

type CommandInput struct {
	cmd Command
	buf string
}

func commandFromPrefix(r rune) Command {
	switch r {
	case ':':
		return CmdJump
	case '/':
		return CmdSearch
	case '#':
		return CmdComment
	default:
		return CmdNone
	}
}

func (m *model) commandBadge(cmd Command) string {
	switch cmd {
	case CmdSearch:
		return "[/]"
	case CmdFilter:
		return "[~]"
	case CmdJump:
		return "[:]"
	case CmdComment:
		return "[#]"
	case CmdMark:
		return "[*!]"
	default:
		return "[-]"
	}
}

func (m *model) commandPrompt(cmd Command) string {
	switch cmd {
	case CmdSearch:
		return "search: "
	case CmdFilter:
		return "regex filter: "
	case CmdJump:
		return "jump to line: "
	case CmdComment:
		return "comment: "
	case CmdMark:
		return "mark: "
	default:
		return ""
	}
}

func (m *model) commandHintsLine(cmd Command) string {
	switch cmd {
	case CmdFilter:
		return "enter: apply esc: cancel (regex is defaulted to case insensitive)"
	case CmdMark:
		return "r/g/a: mark   c: clear   esc: cancel"
	default:
		return "enter: apply   esc: cancel"
	}
}

// activeCommandLine returns the command prompt text for the footer status line.
func (m *model) activeCommandLine() string {
	badge := m.commandBadge(m.ui.command.cmd)
	prompt := m.commandPrompt(m.ui.command.cmd)
	return badge + " " + prompt + m.ui.command.buf
}

func (m *model) enterCommand(cmd Command, seed string, showHint bool, refresh bool) tea.Cmd {
	m.ui.command.cmd = cmd
	if seed != "" {
		m.ui.command.buf = seed
	} else {
		switch cmd {
		case CmdFilter:
			if m.data.filterRegex != nil {
				m.ui.command.buf = m.data.filterPattern
			} else {
				m.ui.command.buf = ""
			}
		case CmdSearch:
			m.ui.command.buf = m.ui.searchQuery
		case CmdComment:
			m.ui.command.buf = m.getCommentContent(m.currentRowHashID())
		default:
			m.ui.command.buf = ""
		}
	}

	m.ui.mode = modeCommand
	if refresh {
		m.refreshView("enter-command", false)
	}
	if showHint {
		return m.startNotice(m.commandHintsLine(cmd), "info", noticeDuration)
	}
	return nil
}

func (m *model) exitCommand(refresh bool) tea.Cmd {
	m.ui.command = CommandInput{}
	m.ui.mode = modeView
	if refresh {
		m.refreshView("exit-command", false)
	}
	return nil
}
