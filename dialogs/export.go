package dialogs

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Messages ---------------------------------------------------------------

type (
	ExportRequestedMsg struct{}
	ExportConfirmedMsg struct{ Path string }
	ExportCanceledMsg  struct{}
	ExportErrorMsg     struct{ Err error }
	ExportOKMsg        struct{ Path string }
)

type Export struct {
	input   textinput.Model
	visible bool
	// optional: remember the last directory
	lastDir string
}

func (d Export) Init() tea.Cmd { return d.input.Focus() }

func NewExportDialog(defaultName, lastDir string) *Export {
	ti := textinput.New()
	// Prompt and placeholder
	ti.Placeholder = defaultName
	ti.Prompt = "Export as: "
	ti.CharLimit = 256
	// Wide enough for typical paths
	ti.Width = 50
	if defaultName != "" {
		ti.SetValue(defaultName)
	}
	return &Export{input: ti, visible: true, lastDir: lastDir}
}

func (d *Export) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	log.Printf("ExportDialog:Update:: Called\n")
	if !d.visible {
		return d, nil
	}
	switch m := msg.(type) {
	case tea.KeyMsg:
		log.Printf("Export:Update::Handle Key Message\n")
		s := m.String()
		switch s {
		case "enter":
			log.Printf("ExportDialog:Update::Enter key was pressed, starting exporting to file.\n")
			val := d.input.Value()
			if val == "" {
				// fall back to placeholder if user left it blank
				val = d.input.Placeholder
			}
			if val == "" {
				return d, nil
			}
			path := val
			// Expand "." to lastDir if provided
			if d.lastDir != "" && !filepath.IsAbs(path) && filepath.Dir(path) == "." {
				path = filepath.Join(d.lastDir, filepath.Base(path))
			}
			return d, func() tea.Msg { return ExportConfirmedMsg{Path: path} }
		case "esc":
			log.Printf("ExportDialog:Update::Esc key was prssed, cancel anything to do with this\n")
			return d, func() tea.Msg { return ExportCanceledMsg{} }
		}
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

func (d Export) View() string {
	if !d.visible {
		return ""
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("252")). // keep your light border
		BorderBackground(lipgloss.Color("236")). // ← match the overlay!
		Padding(1, 2).
		Width(60) // no .Background()

	help := lipgloss.NewStyle().
		Faint(true).
		Render("enter to export • esc to cancel")

	content := fmt.Sprintf("%s\n\n%s", d.input.View(), help)
	return box.Render(content)
}

func (d *Export) Show() {
	d.visible = true
	d.input.Focus()
}

func (d *Export) Hide() {
	d.visible = false
	d.input.Blur()
}

func (d *Export) Focus() tea.Cmd { return d.input.Focus() }
func (d *Export) Blur()          { d.input.Blur() }
func (d Export) IsVisible() bool { return d.visible }
