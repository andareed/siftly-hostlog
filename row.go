package main

import (
	"hash/fnv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type renderedRow struct {
	cols          []string
	height        int
	id            uint64
	originalIndex int // Essentially the row number of the source, not a unique ID
}

// method on the struct
func (r renderedRow) ComputeID() uint64 {
	h := fnv.New64a()
	for _, col := range r.cols {
		norm := strings.ToLower(strings.TrimSpace(col))
		h.Write([]byte(norm))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

func (r *renderedRow) Join(sep string) string {
	var b strings.Builder

	for i, col := range r.cols {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(col)
	}

	return b.String()
}

// String implements fmt.Stringer – choose your default delimiter here.
func (r *renderedRow) String() string {
	// For regex + clipboard, a tab is usually a great default.
	return r.Join("\t")
}

func (r *renderedRow) Render(style lipgloss.Style, colsMeta []ColumnMeta) string {
	var rendered []string

	for i, text := range r.cols {
		if i >= len(colsMeta) {
			break
		}
		meta := colsMeta[i]

		if !meta.Visible || meta.Width <= 0 {
			// Skip hidden / zero-width columns completely
			continue
		}

		cell := style.Width(meta.Width).Render(text)
		rendered = append(rendered, cell)
	}

	joined := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
	r.height = lipgloss.Height(joined)
	return joined
}
