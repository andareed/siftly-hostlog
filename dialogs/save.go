package dialogs

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Messages ---------------------------------------------------------------

type (
	SaveRequestedMsg struct{}
	SaveConfirmedMsg struct{ Path string }
	SaveCanceledMsg  struct{}
	SaveErrorMsg     struct{ Err error }
	SaveOKMsg        struct{ Path string }
)

// --- Key bindings -----------------------------------------------------------

type keymap struct {
	save   key.Binding // Ctrl+S
	saveAs key.Binding // Ctrl+Shift+S
	quit   key.Binding // Ctrl+C / q to quit (example)
}

func defaultKeymap() keymap {
	return keymap{
		save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		saveAs: key.NewBinding(
			key.WithKeys("ctrl+shift+s"),
			key.WithHelp("ctrl+⇧+s", "save as…"),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
	}
}

// --- Save dialog (modal) ----------------------------------------------------

type Save struct {
	input   textinput.Model
	visible bool
	// optional: remember the last directory
	lastDir string
}

func (d Save) Init() tea.Cmd { return d.input.Focus() }

func NewSaveDialog(defaultName, lastDir string) *Save {
	ti := textinput.New()
	// Prompt and placeholder
	ti.Placeholder = defaultName
	ti.Prompt = "Save as: "
	ti.CharLimit = 256
	// Wide enough for typical paths
	ti.Width = 50
	if defaultName != "" {
		ti.SetValue(defaultName)
	}
	return &Save{input: ti, visible: true, lastDir: lastDir}
}

func (d *Save) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	log.Printf("SaveDialog:Update:: Called\n")
	if !d.visible {
		return d, nil
	}
	switch m := msg.(type) {
	case tea.KeyMsg:
		log.Printf("Update::Handle Key Message\n")
		s := m.String()
		switch s {
		case "enter":
			log.Printf("SaveDialog:Update::Enter key was pressed, starting saving to file.\n")
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
			return d, func() tea.Msg { return SaveConfirmedMsg{Path: path} }
		case "esc":
			log.Printf("SaveDialog:Update::Esc key was prssed, cancel anything to do with this\n")
			return d, func() tea.Msg { return SaveCanceledMsg{} }
		}
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

func (d Save) View() string {
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
		Render("enter to save (save.go) • esc to cancel")

	content := fmt.Sprintf("%s\n\n%s", d.input.View(), help)
	return box.Render(content)
}

func (d *Save) Show() {
	d.visible = true
	d.input.Focus()
}

func (d *Save) Hide() {
	d.visible = false
	d.input.Blur()
}

func (d *Save) Focus() tea.Cmd { return d.input.Focus() }
func (d *Save) Blur()          { d.input.Blur() }
func (d Save) IsVisible() bool { return d.visible }

// --- App model --------------------------------------------------------------

// type model struct {
// 	keys     keymap
// 	content  string // pretend this is your document buffer
// 	filename string // current file path (empty until saved)

// 	// modal state
// 	showingSaveAs bool
// 	dialog        saveDialog

// 	status string
// }
