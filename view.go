package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/andareed/siftly-hostlog/logging"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func (m *model) headerView() string {
	// Width for row numbers + pill + comment markers
	markerWidth := len(fmt.Sprintf("%d", len(m.data.rows))) +
		utf8.RuneCountInString(pillMarker) +
		utf8.RuneCountInString(commentMarker)

	var cells []string

	for _, col := range m.data.header {
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
		MarksOnly:     m.data.showOnlyMarked,
		Row:           m.cursor + 1,
		TotalRows:     len(m.data.filteredIndices),
		StatusMessage: "",
		Legend:        "(? help · f filter · / search · t time window · c edit comment · v view comments)",
	}
	if m.data.filterRegex != nil && m.data.filterRegex.String() != "" {
		st.FilterLabel = m.data.filterRegex.String()
	}
	if m.ui.noticeMsg != "" {
		st.StatusMessage = noticeText(m.ui.noticeMsg, m.ui.noticeType)
	}
	if st.StatusMessage == "" {
		st.StatusMessage = m.timeWindowStatusLabel()
	}

	if logging.IsDebugMode() {
		debug := fmt.Sprintf(" dbg term=%dx%d vp=%dx%d cur=%d vis=%d-%d page=%d ch=%d hf=%d abv=%d",
			m.terminalWidth, m.terminalHeight, m.viewport.Width, m.viewport.Height,
			m.cursor, m.ui.visibleStart, m.ui.visibleEnd, m.pageRowSize,
			m.ui.debugCursorHeight, m.ui.debugHeightFree, m.ui.debugDesiredAboveHeight,
		)
		st.Legend = st.Legend + " |" + debug
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
	if m.ui.timeWindow.open {
		parts = append(parts, m.timeWindowDrawerView(contentW))
	}
	parts = append(parts, m.footerView(contentW)) // always
	return appstyle.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *model) renderRowAt(filteredIdx int) (string, int, bool) {
	if filteredIdx < 0 || filteredIdx >= len(m.data.filteredIndices) {
		return "", 0, false
	}

	selected := filteredIdx == m.cursor
	rowBgStyle := rowStyle
	rowPrefix := bgSeq(lipgloss.Color("")) + fgSeq(lipgloss.Color(rowTextFGColor))
	if selected {
		rowBgStyle = rowSelectedStyle
		rowPrefix = bgSeq(lipgloss.Color(rowSelectedBGColor)) + fgSeq(lipgloss.Color(rowSelectedTextFGColor))
	}
	rowSuffix := termenv.CSI + "0m"

	rowIdx := m.data.filteredIndices[filteredIdx]
	rowPtr := &m.data.rows[rowIdx]
	row := *rowPtr

	_, commentPresent := m.data.commentRows[row.id]
	standardMarker := m.getRowMarker(row.id)

	// figure out how wide the row number gutter needs to be
	markerWidth := len(fmt.Sprintf("%d", len(m.data.rows))) + utf8.RuneCountInString(commentMarker)

	// Standard mark seems to reset any bg colour attempts to need to render anything preceding it
	firstLineMarker := standardMarker + rowBgStyle.Render(fmt.Sprintf("%*d", markerWidth, row.originalIndex))
	additionalLineMarker := standardMarker + rowBgStyle.Render(strings.Repeat(" ", markerWidth))

	if commentPresent {
		firstLineMarker = standardMarker + rowBgStyle.Render(commentMarker+fmt.Sprintf("%*d", markerWidth-utf8.RuneCountInString(commentMarker), row.originalIndex))
	}

	contentRow := row
	if m.ui.searchQuery != "" {
		cols := make([]string, len(row.cols))
		for i, col := range row.cols {
			cols[i] = highlightMatches(col, m.ui.searchQuery)
		}
		contentRow.cols = cols
	}
	content := contentRow.Render(cellStyle, m.data.header)
	rowPtr.height = contentRow.height
	lines := strings.Split(content, "\n")

	for i := range lines {
		left := additionalLineMarker
		line := lines[i]
		if m.ui.searchQuery != "" {
			line = restoreRowStyleAfterReset(line, rowPrefix)
		}
		right := rowPrefix + line + rowSuffix
		if i == 0 { // first line
			left = firstLineMarker
		}
		lines[i] = left + right
	}

	rendered := strings.Join(lines, "\n")
	return rendered, contentRow.height, true
}

func highlightMatches(text string, query string) string {
	q := strings.TrimSpace(query)
	if q == "" || text == "" {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(q)
	var b strings.Builder
	start := 0
	for {
		idx := strings.Index(lowerText[start:], lowerQuery)
		if idx == -1 {
			b.WriteString(text[start:])
			break
		}
		idx += start
		b.WriteString(text[start:idx])
		match := text[idx : idx+len(lowerQuery)]
		b.WriteString(searchHighlight.Render(match))
		start = idx + len(lowerQuery)
	}
	return b.String()
}

func restoreRowStyleAfterReset(s string, rowPrefix string) string {
	if rowPrefix == "" {
		return s
	}
	reset := termenv.CSI + "0m"
	if !strings.Contains(s, reset) {
		return s
	}
	return strings.ReplaceAll(s, reset, reset+rowPrefix)
}

func fgSeq(c lipgloss.Color) string {
	return colorSeq(c, false)
}

func bgSeq(c lipgloss.Color) string {
	return colorSeq(c, true)
}

func colorSeq(c lipgloss.Color, bg bool) string {
	value := string(c)
	if value == "" {
		if bg {
			return termenv.CSI + "49m"
		}
		return termenv.CSI + "39m"
	}
	profile := lipgloss.ColorProfile()
	tc := profile.Color(value)
	if tc == nil {
		return ""
	}
	return termenv.CSI + tc.Sequence(bg) + "m"
}

func (m *model) getRowMarker(index uint64) string {
	switch m.data.markedRows[index] {
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

	if len(m.data.filteredIndices) == 0 && cursor < 0 {
		logging.Debugf("renderTable: Returning blank filteredIndices Lenght[%d] cursor[%d]", len(m.data.filteredIndices), cursor)

		return ""
	}
	//TODO: Defect here as we should be using the row count not the display index to maintain between a filter and non-filtered list
	if len(m.data.filteredIndices) < cursor {
		m.cursor = 0
		cursor = 0
	}
	renderedRows, startIdx, endIdx := m.computeVisibleRows(cursor, viewportHeight)
	m.ui.visibleStart = startIdx
	m.ui.visibleEnd = endIdx
	// Metrics
	m.pageRowSize = len(renderedRows)
	m.lastVisibleRowCount = len(renderedRows)

	// Combine rendered rows into a string with proper vertical order
	var b strings.Builder
	for _, r := range renderedRows {
		b.WriteString(r + "\n")
	}

	return b.String()
}

func (m *model) computeVisibleRows(cursor int, viewportHeight int) ([]string, int, int) {
	cursorRenderedRow, cursorHeight, ok := m.renderRowAt(cursor)
	if !ok {
		return nil, 0, 0
	}

	heightFree := viewportHeight - cursorHeight
	desiredAboveHeight := heightFree / 2
	if desiredAboveHeight < 0 {
		desiredAboveHeight = 0
	}
	m.ui.debugCursorHeight = cursorHeight
	m.ui.debugHeightFree = heightFree
	m.ui.debugDesiredAboveHeight = desiredAboveHeight
	upIndex := cursor - 1
	downIndex := cursor + 1

	var above []string
	var below []string

	aboveHeight := 0
	for heightFree > 0 && (upIndex >= 0 || downIndex < len(m.data.filteredIndices)) {
		if upIndex >= 0 && aboveHeight < desiredAboveHeight {
			rendered, height, ok := m.renderRowAt(upIndex)
			if ok && height <= heightFree {
				above = append(above, rendered)
				heightFree -= height
				aboveHeight += height
				upIndex--
				continue
			}
		}
		if downIndex < len(m.data.filteredIndices) {
			rendered, height, ok := m.renderRowAt(downIndex)
			if ok && height <= heightFree {
				below = append(below, rendered)
				heightFree -= height
				downIndex++
				continue
			}
		}
		if upIndex >= 0 {
			rendered, height, ok := m.renderRowAt(upIndex)
			if ok && height <= heightFree {
				above = append(above, rendered)
				heightFree -= height
				aboveHeight += height
				upIndex--
				continue
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

	startIdx := cursor - len(above)
	endIdx := cursor + len(below)
	return renderedRows, startIdx, endIdx
}
