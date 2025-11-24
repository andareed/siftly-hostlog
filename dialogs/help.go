package dialogs

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Messages ---------------------------------------------------------------

type (
	HelpRequestedMsg struct{}
	HelpCanceledMsg  struct{}
	HelpErrorMsg     struct{ Err error }
	HelpOKMsg        struct{ Path string } // kept for compatibility, but not used now
)

// Help is now just a visible flag + a list of key bindings to show.
type Help struct {
	visible  bool
	bindings []key.Binding
}

func (d Help) Init() tea.Cmd { return nil }

// NewHelpDialog creates a new help dialog showing the given bindings.
func NewHelpDialog(bindings []key.Binding) *Help {
	return &Help{
		visible:  true,
		bindings: bindings,
	}
}

func (d *Help) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	log.Printf("HelpDialog:Update:: Called\n")

	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "enter", "esc":
			d.visible = false
			// If you want to send a message back to the parent:
			// return d, func() tea.Msg { return HelpCanceledMsg{} }
			return d, nil
		}
	}

	return d, nil
}

func (d Help) View() string {
	if !d.visible {
		return ""
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("252")). // keep your light border
		BorderBackground(lipgloss.Color("236")). // match the overlay
		Padding(1, 2).
		Width(60)

	// Build lines "keys   description" from the bindings.
	var lines []string
	for _, b := range d.bindings {
		helpItem := b.Help()
		keys, desc := helpItem.Key, helpItem.Desc
		line := fmt.Sprintf("%-12s %s", keys, desc)
		lines = append(lines, line)
	}

	helpHint := lipgloss.NewStyle().
		Faint(true).
		Render("enter/esc to return")

	content := fmt.Sprintf("%s\n\n%s", strings.Join(lines, "\n"), helpHint)
	return box.Render(content)
}

func (d *Help) Show() {
	d.visible = true
}

func (d *Help) Hide() {
	d.visible = false
}

func (d *Help) Focus() tea.Cmd { return nil }
func (d *Help) Blur()          {}
func (d Help) IsVisible() bool { return d.visible }
