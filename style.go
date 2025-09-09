package main

import "github.com/charmbracelet/lipgloss"

var (
	// Styles
	appstyle       = lipgloss.NewStyle().Margin(1, 2)
	headerStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("254")).Padding(0, 0)
	markedRedStyle = lipgloss.NewStyle().Background(lipgloss.Color("124")).Foreground(lipgloss.Color("254")).Padding(0, 1)
	cellStyle      = lipgloss.NewStyle().Padding(0, 1)
	markedStyle    = lipgloss.NewStyle()
	markedRowStyle = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	inputStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1)
	tableStyle     = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	redMarker      = lipgloss.NewStyle().Background(lipgloss.Color("1")).Foreground(lipgloss.Color("1")).Render("▐")
	greenMarker    = lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("2")).Render("▐")
	amberMarker    = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("3")).Render("▐")
	defaultMarker  = " " // or "▐" with neutral color if preferred
)

// func (r *renderedRow) Height() int {
