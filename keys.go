package main

import (
	"github.com/charmbracelet/bubbles/key"
)

type Keymap struct {
	Quit          key.Binding
	MarkMode      key.Binding
	ShowMarksOnly key.Binding
	NextMark      key.Binding
	PrevMark      key.Binding
	Filter        key.Binding
	Search        key.Binding
	ClearFilter   key.Binding
	SearchNext    key.Binding
	SearchPrev    key.Binding
	ShowComment   key.Binding
	EditComment   key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	RowDown       key.Binding
	RowUp         key.Binding
	OpenHelp      key.Binding
	ScrollLeft    key.Binding
	ScrollRight   key.Binding
	SaveToFile    key.Binding
	ExportToFile  key.Binding
	CopyRow       key.Binding
	JumpToStart   key.Binding
	JumpToEnd     key.Binding
	JumpToLineNo  key.Binding
}

var Keys = Keymap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "Quit"),
	),
	MarkMode: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "Mark mode"),
	),
	ShowMarksOnly: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "Toggle show only marked"),
	),
	NextMark: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Jump to next mark"),
	),
	PrevMark: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "Jump to previous mark"),
	),
	ShowComment: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "View comments"),
	),
	EditComment: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Edit comment on selected row"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Filter by Regex"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "Search"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "Clear current filter"),
	),
	SearchNext: key.NewBinding(
		key.WithKeys("]", "ctrl+n"),
		key.WithHelp("]/ctrl+n", "Next search"),
	),
	SearchPrev: key.NewBinding(
		key.WithKeys("[", "ctrl+p"),
		key.WithHelp("[/ctrl+p", "Prev search"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("u", "pgup"),
		key.WithHelp("u/pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("d", "pgdown"),
		key.WithHelp("d/pgdown", "Page down"),
	),
	RowDown: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "Move a row down"),
	),
	RowUp: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "Move a row up"),
	),
	OpenHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "Help / keys"),
	),
	ScrollLeft: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h or <- ", "Scroll the grid left"),
	),
	ScrollRight: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l or >- ", "Scroll the grid right"),
	),
	SaveToFile: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Save to filename"),
	),
	ExportToFile: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "Export to filename"),
	),
	CopyRow: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "Copy row to clipboard"),
	),
	JumpToStart: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g/home", "Jump to start"),
	),
	JumpToEnd: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G/end", "Jump to end"),
	),
	JumpToLineNo: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "Jump To line number"),
	),
}

func (k Keymap) Legend() []key.Binding {
	return []key.Binding{
		k.Quit,
		k.MarkMode,
		k.ShowMarksOnly,
		k.NextMark,
		k.PrevMark,
		k.Filter,
		k.Search,
		k.ClearFilter,
		k.SearchNext,
		k.SearchPrev,
		k.EditComment,
		k.ShowComment,
		k.PageUp,
		k.PageDown,
		k.CopyRow,
		k.ExportToFile,
		k.SaveToFile,
		k.RowUp,
		k.RowDown,
		k.JumpToStart,
		k.JumpToEnd,
		k.JumpToLineNo,
	}
}
