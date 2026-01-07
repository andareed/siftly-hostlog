package dialogs

import tea "github.com/charmbracelet/bubbletea"

// Dialog is the common interface all dialogs (Save, Export, Help, etc.) implement.
// It keeps your model logic generic.
type Dialog interface {
	Init() tea.Cmd // optional, can return nil
	Update(msg tea.Msg) (Dialog, tea.Cmd)
	View() string

	Focus() tea.Cmd
	Blur()
	IsVisible() bool
	Show()
	Hide()
}
