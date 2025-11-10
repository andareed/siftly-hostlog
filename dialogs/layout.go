package dialogs

import "github.com/charmbracelet/lipgloss"

func center(s string, width, height int) string {
	box := lipgloss.NewStyle().Width(width).Height(height).Align(lipgloss.Center, lipgloss.Center)
	return box.Render(s)
}
