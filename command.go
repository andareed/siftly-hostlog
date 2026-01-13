package main

import "fmt"

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

func (m *model) idleCommandHintsLine() string {
	return "/ search   f filter   : jump   m mark   # comment"
}

// activeCommandLine returns the command prompt text for the footer status line.
func (m *model) activeCommandLine() string {
	badge := m.commandBadge(m.ci.cmd)
	prompt := m.commandPrompt(m.ci.cmd)
	return badge + " " + prompt + m.ci.buf
}

func (m *model) commandRightContext() string {
	return fmt.Sprintf("%d/%d",
		m.cursor+1,
		len(m.filteredIndices),
	)
}
