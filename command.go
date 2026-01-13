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

func CommandFromPrefix(r rune) Command {
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
		return "[SEARCH]"
	case CmdFilter:
		return "[FILTER]"
	case CmdJump:
		return "[JUMP]"
	case CmdComment:
		return "[COMMENT]"
	case CmdMark:
		return "[MARK]"
	default:
		return "[NORMAL]"
	}
}

func (m *model) commandPrompt(cmd Command) string {
	switch cmd {
	case CmdSearch:
		return "search: "
	case CmdFilter:
		return "filter: "
	case CmdJump:
		return "line: "
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
	case CmdMark:
		return "r/g/a: mark   c: clear   esc: cancel"
	default:
		return "enter: apply   esc: cancel"
	}
}

// activeCommandLine returns the command prompt text for the footer status line.
func (m *model) activeCommandLine() string {
	badge := m.commandBadge(m.ci.cmd)
	prompt := m.commandPrompt(m.ci.cmd)
	return badge + " " + prompt + m.ci.buf
}

func (m *model) enterCommand(cmd Command, seed string, showHint bool, refresh bool) tea.Cmd {
	m.ci.cmd = cmd
	if seed != "" {
		m.ci.buf = seed
	} else {
		switch cmd {
		case CmdFilter:
			if m.filterRegex != nil {
				m.ci.buf = m.filterRegex.String()
			} else {
				m.ci.buf = ""
			}
		case CmdSearch:
			if m.searchRegex != nil {
				m.ci.buf = m.searchRegex.String()
			} else {
				m.ci.buf = ""
			}
		default:
			m.ci.buf = ""
		}
	}

	m.currentMode = modeCommand
	if refresh {
		m.refreshView("enter-command", false)
	}
	if showHint {
		return m.startNotice(m.commandHintsLine(cmd), "info", noticeDuration)
	}
	return nil
}

func (m *model) exitCommand(refresh bool) tea.Cmd {
	m.ci = CommandInput{}
	m.currentMode = modeView
	if refresh {
		m.refreshView("exit-command", false)
	}
	return nil
}
