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
	ClearFilter   key.Binding
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
}

var Keys = Keymap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	MarkMode: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "mark mode"),
	),
	ShowMarksOnly: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "toggle show only marked"),
	),
	NextMark: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next mark"),
	),
	PrevMark: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "previous mark"),
	),
	ShowComment: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Show Comments"),
	),
	EditComment: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "Edit comment on selected row"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "clear filter"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("u", "pgup"),
		key.WithHelp("u/pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("d", "pgdown"),
		key.WithHelp("d/pgdown", "page down"),
	),
	RowDown: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "move down"),
	),
	RowUp: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "move up"),
	),
	OpenHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help / keys"),
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
		key.WithHelp("ctrl+c", "copy row to clipboard"),
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
		k.ClearFilter,
		k.EditComment,
		k.ShowComment,
		k.PageUp,
		k.PageDown,
		k.CopyRow,
	}
}
