package main

import "github.com/charmbracelet/lipgloss"

var (
	// Styles
	appstyle = lipgloss.NewStyle().Margin(1, 2)
	//headerStyle      = lipgloss.NewStyle().Bold(true).Padding(0, 0)
	headerStyle = lipgloss.NewStyle().BorderStyle(lipgloss.Border{
		Left:  " ",
		Right: " ",
	}).BorderLeft(true).BorderRight(true)
	rowStyle         = lipgloss.NewStyle()
	rowSelectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("#3a3a3a"))

	// Row Text (no background)
	rowTextStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#c0c0c0"))
	rowSelectedTextstyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))

	// selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("254")).Padding(0, 0)
	// markedRedStyle = lipgloss.NewStyle().Background(lipgloss.Color("124")).Foreground(lipgloss.Color("254")).Padding(0, 1)
	cellStyle = lipgloss.NewStyle().Padding(0, 1)
	// markedStyle    = lipgloss.NewStyle()
	// markedRowStyle = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	// helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	inputStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1)
	tableStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	redMarker     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	greenMarker   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	amberMarker   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	defaultMarker = " " // defaultMarker is used to replace pillMarker when no RAG has been marked agaist a record
	pillMarker    = "▐"
	commentMarker = "[*]"

	commentArea = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("245")). // subtle gray
			Padding(0, 0).BorderLeft(true)
)

// func (r *renderedRow) Height() int {
