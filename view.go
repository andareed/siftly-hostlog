package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/andareed/siftly-hostlog/logging"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) headerView() string {
	// Width for row numbers + pill + comment markers
	markerWidth := len(fmt.Sprintf("%d", len(m.rows))) +
		utf8.RuneCountInString(pillMarker) +
		utf8.RuneCountInString(commentMarker)

	var cells []string

	for _, col := range m.header {
		if !col.Visible || col.Width <= 0 {
			continue
		}

		cell := cellStyle.Width(col.Width).Render(col.Name)
		cells = append(cells, cell)
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

	return headerStyle.Render(
		strings.Repeat(" ", markerWidth) + headerRow,
	)
}

// footerView renders the 2-line footer using local (function-scoped) styles/state.
// width is the terminal width (e.g. m.width from tea.WindowSizeMsg).
func (m *model) footerView(width int) string {
	logging.Debugf("footerView mode=%d cmd=%d", m.ui.mode, m.ui.command.cmd)
	styles := defaultFooterStyles()

	footerMode := CmdNone
	modeInput := ""
	switch m.ui.mode {
	case modeView:
		footerMode = CmdNone
	case modeComment:
		footerMode = CmdComment
	case modeCommand:
		switch m.ui.command.cmd {
		case CmdJump:
			footerMode = CmdJump
		case CmdFilter:
			footerMode = CmdFilter
		case CmdSearch:
			footerMode = CmdSearch
		case CmdComment:
			footerMode = CmdComment
		case CmdMark:
			footerMode = CmdMark
		default:
			footerMode = CmdNone
		}
		modeInput = m.activeCommandLine()
	}

	st := footerState{
		Mode:          footerMode,
		ModeInput:     modeInput,
		FileName:      defaultSaveName(*m),
		FilterLabel:   "None",
		MarksOnly:     m.showOnlyMarked,
		Row:           m.cursor + 1,
		TotalRows:     len(m.filteredIndices),
		StatusMessage: "",
		Legend:        "(? help · f filter · c comment)",
	}
	if m.filterRegex != nil && m.filterRegex.String() != "" {
		st.FilterLabel = m.filterRegex.String()
	}
	if m.ui.noticeMsg != "" {
		st.StatusMessage = noticeText(m.ui.noticeMsg, m.ui.noticeType)
	}

	return renderFooter(width, st, styles)
}

func (m *model) View() string {
	if !m.ready {
		return "loading..."
	}

	if m.activeDialog != nil && m.activeDialog.IsVisible() {
		w, h := m.terminalWidth, m.terminalHeight
		return lipgloss.Place(
			w, h,
			lipgloss.Center, lipgloss.Center,
			m.activeDialog.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("236")),
		)
	}

	bordered := tableStyle.Render(m.viewport.View())
	contentW := lipgloss.Width(bordered)

	parts := []string{m.headerView(), bordered}
	if m.ui.drawerOpen {
		parts = append(parts, commentArea.Render(m.drawerPort.View()))
	}
	parts = append(parts, m.footerView(contentW)) // always
	return appstyle.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *model) renderRowAt(filteredIdx int) (string, int, bool) {
	if filteredIdx < 0 || filteredIdx >= len(m.filteredIndices) {
		return "", 0, false
	}

	rowBgStyle := rowStyle
	rowFgStyle := rowTextStyle
	if filteredIdx == m.cursor {
		rowBgStyle = rowSelectedStyle
		rowFgStyle = rowSelectedTextstyle
	}

	rowIdx := m.filteredIndices[filteredIdx]
	row := m.rows[rowIdx]

	_, commentPresent := m.commentRows[row.id]
	standardMarker := m.getRowMarker(row.id)

	// figure out how wide the row number gutter needs to be
	markerWidth := len(fmt.Sprintf("%d", len(m.rows))) + utf8.RuneCountInString(commentMarker)

	// Standard mark seems to reset any bg colour attempts to need to render anything preceding it
	firstLineMarker := standardMarker + rowBgStyle.Render(fmt.Sprintf("%*d", markerWidth, row.originalIndex))
	additionalLineMarker := standardMarker + rowBgStyle.Render(strings.Repeat(" ", markerWidth))

	if commentPresent {
		firstLineMarker = standardMarker + rowBgStyle.Render(commentMarker+fmt.Sprintf("%*d", markerWidth-utf8.RuneCountInString(commentMarker), row.originalIndex))
	}

	content := row.Render(cellStyle, m.header)
	lines := strings.Split(content, "\n")

	for i := range lines {
		left := additionalLineMarker
		right := rowBgStyle.Render(rowFgStyle.Render(lines[i]))
		if i == 0 { // first line
			left = firstLineMarker
		}
		lines[i] = left + right
	}

	rendered := strings.Join(lines, "\n")
	return rendered, row.height, true
}

func (m *model) getRowMarker(index uint64) string {
	switch m.markedRows[index] {
	case MarkRed:
		return redMarker.Render(pillMarker)
	case MarkGreen:
		return greenMarker.Render(pillMarker)
	case MarkAmber:
		return amberMarker.Render(pillMarker)
	default:
		return defaultMarker
	}
}

func (m *model) renderViewport() string {
	logging.Debug("renderTable called")
	viewportHeight := m.viewport.Height
	viewPortWidth := m.viewport.Width
	_ = viewPortWidth // TODO: I'm going to use this just need to remember why and where

	cursor := m.cursor

	if len(m.filteredIndices) == 0 && cursor < 0 {
		logging.Debugf("renderTable: Returning blank filteredIndices Lenght[%d] cursor[%d]", len(m.filteredIndices), cursor)

		return ""
	}
	//TODO: Defect here as we should be using the row count not the display index to maintain between a filter and non-filtered list
	if len(m.filteredIndices) < cursor {
		m.cursor = 0
		cursor = 0
	}
	renderedRows, rowCount := m.computeVisibleRows(cursor, viewportHeight)
	// Metrics
	m.pageRowSize = rowCount
	m.lastVisibleRowCount = len(renderedRows)

	// Combine rendered rows into a string with proper vertical order
	var b strings.Builder
	for _, r := range renderedRows {
		b.WriteString(r + "\n")
	}

	return b.String()
}

func (m *model) computeVisibleRows(cursor int, viewportHeight int) ([]string, int) {
	cursorRenderedRow, cursorHeight, ok := m.renderRowAt(cursor)
	if !ok {
		return nil, 0
	}

	heightFree := viewportHeight - cursorHeight
	upIndex := cursor - 1
	downIndex := cursor + 1
	rowCount := 0

	var above []string
	var below []string

	nextAbove := true
	for heightFree > 0 && (upIndex >= 0 || downIndex < len(m.filteredIndices)) {
		if nextAbove {
			if upIndex >= 0 {
				rendered, height, ok := m.renderRowAt(upIndex)
				if ok && height <= heightFree {
					above = append(above, rendered)
					heightFree -= height
					upIndex--
					rowCount++
					nextAbove = false
					continue
				}
			}
			if downIndex < len(m.filteredIndices) {
				rendered, height, ok := m.renderRowAt(downIndex)
				if ok && height <= heightFree {
					below = append(below, rendered)
					heightFree -= height
					downIndex++
					rowCount++
					nextAbove = true
					continue
				}
			}
		} else {
			if downIndex < len(m.filteredIndices) {
				rendered, height, ok := m.renderRowAt(downIndex)
				if ok && height <= heightFree {
					below = append(below, rendered)
					heightFree -= height
					downIndex++
					rowCount++
					nextAbove = true
					continue
				}
			}
			if upIndex >= 0 {
				rendered, height, ok := m.renderRowAt(upIndex)
				if ok && height <= heightFree {
					above = append(above, rendered)
					heightFree -= height
					upIndex--
					rowCount++
					nextAbove = false
					continue
				}
			}
		}
		break
	}

	renderedRows := make([]string, 0, len(above)+1+len(below))
	for i := len(above) - 1; i >= 0; i-- {
		renderedRows = append(renderedRows, above[i])
	}
	renderedRows = append(renderedRows, cursorRenderedRow)
	renderedRows = append(renderedRows, below...)

	return renderedRows, rowCount
}
