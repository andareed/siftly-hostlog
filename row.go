package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var columnWeights = []float64{
	0.1,  // Time
	0.15, // Host
	0.4,  // Details
	0.1,  // Appliance
	0.15, // MAC
	0.1,  // IPv6
	0,
}

type renderedRow struct {
	cols   []string
	height int
}

func (r renderedRow) String() string {
	var rowAsString strings.Builder

	for _, col := range r.cols {
		rowAsString.WriteString(col)
		rowAsString.WriteString("")
	}
	return rowAsString.String()
}

// func (r *renderedRow) Render() string {
// return lipgloss.JoinHorizontal(lipgloss.Top, r.cols...)
// }

func (r *renderedRow) Render(style lipgloss.Style, rowWidth int, columnWeights []float64) string {
	rendered := make([]string, len(r.cols))
	for i, col := range r.cols {
		colWidth := int(float64(rowWidth) * columnWeights[i])
		rendered[i] = style.Width(colWidth).Render(col)
	}
	joined := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

	r.height = lipgloss.Height(joined) // store it
	// r.cached = joined                  // optionally cache it

	return joined
}
