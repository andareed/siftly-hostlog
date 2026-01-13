package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/andareed/siftly-hostlog/clipboard"
	"github.com/andareed/siftly-hostlog/dialogs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//var _ tea.Msg = dialogs.SaveRequestedMsg{}

type mode int

const (
	modeView mode = iota
	modeFilter
	modeMarking
	modeComment
	modeCommand
)

//TODO Replace the name renderedRow, as these are not rendered anymore

type model struct {
	header              []ColumnMeta // single row for column titles in headerview
	rows                []renderedRow
	viewport            viewport.Model
	drawerPort          viewport.Model
	drawerOpen          bool
	drawerHeight        int
	ready               bool
	cursor              int // index into rows
	lastVisibleRowCount int
	currentMode         mode
	markedRows          map[uint64]MarkColor // map row index to color code
	commentRows         map[uint64]string    // map row index to string to store comments
	showOnlyMarked      bool
	filterRegex         *regexp.Regexp
	searchRegex         *regexp.Regexp
	filteredIndices     []int // to store the list of indicides that match the current regex
	filterInput         textinput.Model
	commentInput        textarea.Model
	terminalHeight      int
	terminalWidth       int
	pageRowSize         int
	noticeMsg           string
	noticeType          string
	noticeSeq           int
	activeDialog        dialogs.Dialog
	fileName            string // filename the data will be saved to
	InitialPath         string
	lastExportFileName  string
	ci                  CommandInput
}

func (m *model) InitialiseUI() {
	fi := textinput.New()
	fi.Placeholder = "Regex Filter..."
	//fi.Focus()
	fi.CharLimit = 156
	fi.Width = 50

	ca := textarea.New()
	ca.Placeholder = "Comment:"
	//ca.Focus()
	ca.CharLimit = 256

	m.filterInput = fi
	m.commentInput = ca

	m.showOnlyMarked = false
	m.drawerPort = viewport.New(0, 0)
	m.drawerHeight = 13 // TODO:should be a better way of calcing this rather than hardcoding
	m.drawerOpen = false
}

func (m *model) Init() tea.Cmd {
	m.applyFilter()
	log.Println("siftly-hostlog: Initialised")
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		log.Printf("KEY: %q type=%v mode=%v dialog=%T visible=%v",
			km.String(), km.Type, m.currentMode,
			m.activeDialog, m.activeDialog != nil && m.activeDialog.IsVisible(),
		)
	}

	// Might be messy but all tick related messages need to be handle first and discard if necessary
	switch fmt.Sprintf("%T", msg) {
	case "cursor.BlinkMsg", "cursor.BlinkCanceledMsg":
		return m, nil // Swallow the sound of the constant Blink Msgs
	}
	log.Printf("model:Update called with msg: %#T: %#v\n", msg, msg)

	if m.activeDialog != nil && m.activeDialog.IsVisible() {
		if km, ok := msg.(tea.KeyMsg); ok {
			log.Printf("DIALOG UPDATE: %T got key %q\n", m.activeDialog, km.String())
		}
		log.Printf("model:Update:: Dialog box is active forward update to it\n")
		if _, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.activeDialog, cmd = m.activeDialog.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case clearNoticeMsg:
		if msg.id == m.noticeSeq {
			m.noticeMsg = ""
			m.noticeType = ""
		}
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width
		m.viewport = viewport.New(0, 0) // TODO: PRetty sure this is redundant
		m.recomputeLayout(m.terminalHeight, m.terminalWidth)
		m.ready = true
		m.viewport.SetContent(m.renderTable())
		return m, nil
	case dialogs.HelpRequestedMsg:
		log.Printf("Updated was called with msg HelpRequestedMsg (should pop a help dialog box)\n")
		m.activeDialog = dialogs.NewHelpDialog(Keys.Legend())
		m.activeDialog.Show()
	case dialogs.SaveRequestedMsg:
		log.Printf("Update was called with msg SaveRequestedMsg (should pop a dialog box)\n")
		m.activeDialog = dialogs.NewSaveDialog(defaultSaveName(*m), filepath.Dir(m.fileName))
		m.activeDialog.Show()
	case dialogs.SaveConfirmedMsg:
		log.Printf("model:Update::Received SaveConfirmedMsg saving to the current file using path %q\n", msg.Path)
		m.activeDialog.Hide()
		if err := SaveModel(m, msg.Path); err != nil {
			cmd := m.startNotice("Error", "", noticeDuration)
			return m, cmd
		}
		cmd := m.startNotice("Saved succeeded", "", noticeDuration)
		m.fileName = msg.Path
		return m, cmd
	case dialogs.SaveCanceledMsg:
		log.Printf("model:Update::Received SaveCanceledMsg from dialog and hiding the active dialog\n")
		m.activeDialog.Hide()
	case dialogs.ExportRequestedMsg:
		log.Printf("Update wwas called with msg ExportRequestMsg (pop dialog for exports)\n")
		m.activeDialog = dialogs.NewExportDialog(defaultExportName(*m), filepath.Dir(m.fileName))
		// TODO: What filename should this default to for an export?
		m.activeDialog.Show()
	case dialogs.ExportConfirmedMsg:
		// Screen Select should be exported as a csv
		log.Printf("module:Update::ExportConfirmedMsg begin exporting to the file\n")
		m.activeDialog.Hide()
		// TODO: Insert call for exporting current selection to the csv
		if err := ExportModel(m, msg.Path); err != nil {
			cmd := m.startNotice("Export Error", "", noticeDuration)
			return m, cmd
		}
		cmd := m.startNotice("Exported succeeded", "", noticeDuration)
		m.lastExportFileName = msg.Path
		return m, cmd
	case dialogs.ExportCanceledMsg:
		log.Printf("model:Update:: Received ExportCanceledMsg, close down the dialog\n")
		m.activeDialog.Hide()
	}

	return m, nil
}

func (m *model) recomputeLayout(height int, width int) {
	// Computes the layout based on whats being rendered
	log.Printf("recomputeLayout called with height[%d] width[%d]\n", height, width)
	height -= 6 // TODO understand this better here? 2 is for the footer, header, plus borders probably
	width -= 6
	if m.drawerOpen {
		drawerContentHeight := 8
		m.drawerHeight = drawerContentHeight + 2
		height -= m.drawerHeight
		m.drawerPort.Width = width // Minus out the padding.
		m.commentInput.SetWidth(width)
		m.commentInput.SetHeight(drawerContentHeight)
		m.drawerPort.Height = drawerContentHeight
		m.drawerPort.Width = width
	}
	log.Printf("Update Received of type Windows Size Message."+
		" ViewPort was [%d] and is now getting set to height[%d] width [%d]/n", m.viewport.Height, height, width)
	m.viewport.Height = height
	m.viewport.Width = width
	m.header = layoutColumns(m.header, width)
}

func (m *model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// if m.activeDialog != nil && m.activeDialog.IsVisible() {
	//
	// return m.handleDialogKey(msg)
	// }
	switch m.currentMode {
	case modeView:
		return m.handleViewModeKey(msg)
	case modeCommand:
		return m.handleCommandKey(msg)
	}

	return m, nil
}

func (m *model) handleViewModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	startCommand := func(c Command) tea.Cmd {
		m.ci.cmd = c
		m.ci.buf = ""
		m.currentMode = modeCommand
		return m.startNotice(m.commandHintsLine(c), "info", noticeDuration)
	}

	switch {
	// Migrating to a command / input method
	case key.Matches(msg, Keys.JumpToLineNo):
		log.Println("Enabling Command: Jumping to specific line number if it exists")
		cmd = startCommand(CmdJump)
	case key.Matches(msg, Keys.Filter):
		log.Println("Enabling Command: Filtering")
		cmd = startCommand(CmdFilter)
	case key.Matches(msg, Keys.MarkMode):
		log.Println("Enable COmmand: Marking")
		cmd = startCommand(CmdMark)
	case key.Matches(msg, Keys.EditComment):
		log.Println("Enable COmmand: Edit Comment")
		cmd = startCommand(CmdComment)
	//TODO: Implement Serach
	case key.Matches(msg, Keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, Keys.CopyRow):
		log.Println("Key Combination for CopyRow To Clipboard")
		cmd = m.copyRowToClipboard()
	case key.Matches(msg, Keys.JumpToStart):
		log.Println("Jumping to start (if filtered will be first row in filter")
		m.jumpToStart()
	case key.Matches(msg, Keys.JumpToEnd):
		log.Println("Jumping to end (if filtered will be last row in filter")
		m.jumpToEnd()
	case key.Matches(msg, Keys.ShowMarksOnly):
		// Show Marks only
		log.Println("Toggle for Show Marks Only has been pressed")
		m.showOnlyMarked = !m.showOnlyMarked
		cmd = m.startNotice(fmt.Sprintf("'Show Only Marked Rows' toggled {%b}", m.showOnlyMarked), "", noticeDuration)
		m.applyFilter()
	case key.Matches(msg, Keys.NextMark):
		// Next mark jump
		log.Println("Here we go; jumping to the next mark")
		m.jumpToNextMark()
	case key.Matches(msg, Keys.PrevMark):
		log.Println("Back once again: jumping to the previous mark")
		m.jumpToPreviousMark()
		m.ready = true
	case key.Matches(msg, Keys.ClearFilter):
		// Clear Filter
		log.Println("Shift F, clearing Filter")
		m.setFilterPattern("") // Set the filter to nothing which will clear
		cmd = m.startNotice("Cleared filter", "", noticeDuration)
	// case key.Matches(msg, Keys.EditComment):
	// 	// Comment (Edit) if the drawer is open (i.e. C has been pressed previously)
	// 	if m.drawerOpen {
	// 		m.commentInput.Focus()
	// 		m.currentMode = modeComment
	// 	}
	case key.Matches(msg, Keys.ShowComment):
		// Comment in Drawer to be toggled opened / closed
		m.drawerOpen = !m.drawerOpen
		log.Printf("handleViewModeKey: Toggling Drawer (bottom view above the footer) now see to [%t]", m.drawerOpen)
		m.recomputeLayout(m.terminalHeight, m.terminalWidth)
	case key.Matches(msg, Keys.RowDown):
		// log.Printf("handleViewModekey: Down or J pressed, moving cursor one position. Cursor [%d] Rows_Total [%d] DrawerOpen[%t]\n", m.cursor, len(m.rows), m.drawerOpen)
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case key.Matches(msg, Keys.RowUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, Keys.PageUp):
		m.pageUp()
	case key.Matches(msg, Keys.PageDown):
		m.pageDown()
	case key.Matches(msg, Keys.OpenHelp):
		return m, func() tea.Msg { return dialogs.HelpRequestedMsg{} }
	case key.Matches(msg, Keys.ScrollLeft):
		m.viewport.ScrollLeft(4) // tune step
	case key.Matches(msg, Keys.ScrollRight):
		m.viewport.ScrollRight(4)
	case key.Matches(msg, Keys.SaveToFile):
		return m, func() tea.Msg { return dialogs.SaveRequestedMsg{} }
	case key.Matches(msg, Keys.ExportToFile):
		return m, func() tea.Msg { return dialogs.ExportRequestedMsg{} }
	}

	//TODO: DON'T THINK WE SHOULD BE RENDERING TABLE EVERY TIME TBH
	if m.ready {
		m.viewport.SetContent((m.renderTable()))
	}
	return m, cmd
}

func (m *model) copyRowToClipboard() tea.Cmd {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		row := m.rows[m.cursor]
		text := row.Join("\t") // Tab Delimetered string
		if err := clipboard.Copy(text); err != nil {
			return m.startNotice("Error with Clipboard occurred.", "", noticeDuration)
		} else {
			return m.startNotice("Copied Row COntent to Clipboard", "", noticeDuration)
		}
	}
	return nil
}

func (m *model) pageDown() {
	if m.cursor+m.pageRowSize < len(m.rows) {
		m.cursor += m.pageRowSize
	} else {
		m.cursor = len(m.rows) - 1
	}
}

func (m *model) pageUp() {
	m.cursor -= m.pageRowSize
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *model) currentRowHashID() uint64 {
	if len(m.filteredIndices) == 0 || m.cursor < 0 || m.cursor >= len(m.filteredIndices) {
		log.Printf("currentRowHashID called but no filteredIndices available (cursor=%d, len=%d)", m.cursor, len(m.filteredIndices))
		return 0 // or some sentinel value
	}
	rowIdx := m.filteredIndices[m.cursor]
	hashId := m.rows[rowIdx].id
	log.Printf("currentRowHashID called returning HashID[%d] for cursor[%d] at filteredIndex[%d] which maps to rowIndex[%d]", hashId, m.cursor, rowIdx, rowIdx)
	return hashId
}

func (m *model) jumpToHashID(hashId uint64) {
	if hashId == 0 {
		log.Printf("jumpToHashID called with HashID of 0, so returning")
		m.cursor = 0
		return
	}

	log.Printf("jumpToHashID called looking for HashID[%d]", hashId)
	for i, idx := range m.filteredIndices {
		log.Printf("jumpToHashID: Checking index[%d] with HashID[%d] against target HashID[%d]", idx, m.rows[idx].id, hashId)
		if m.rows[idx].id == hashId {
			m.cursor = i
			log.Printf("jumpToHashID: Jumping to index [%d] for hashID[%d]", i, hashId)
			return
		}
	}

	m.cursor = 0
	log.Printf("jumpToHashID: No match found for hashID[%d] so setting cursor to 0", hashId)
}

func (m *model) checkViewPortHasData() bool {
	if len(m.filteredIndices) == 0 {
		log.Println("filterIndicies is empty")
		return false
	}
	if m.cursor < 0 {
		log.Println("Cursor at 0 or below")
		return false
	}
	return true
}
func (m *model) applyFilter() {
	log.Printf("applyFilter called")
	// Remember the Hash of what we have current selected

	currentRowHash := m.currentRowHashID()    // should be called before we reset the filteredIndices
	m.filteredIndices = m.filteredIndices[:0] // reset slice

	if m.filterRegex == nil && !m.showOnlyMarked {
		log.Println("applyFilter: No filter text and showOnly marked is false there all indices being added to filteredIncidices")
		// Maybe used clamp?
		for i := range m.rows {
			m.filteredIndices = append(m.filteredIndices, i)
		}
		if len(m.filteredIndices) == 0 {
			m.cursor = 0
		}
		m.jumpToHashID(currentRowHash)

		m.viewport.SetContent(m.renderTable())
		return
	}

	for i, row := range m.rows {
		log.Printf("applyFilter: Checking row index[%d] with HashID[%d]\n", i, row.id)
		log.Printf("applyFilter: Row content is [%s] and Filter regex is [%v]", row.String(), m.filterRegex)
		if m.includeRow(row) {
			log.Printf("Row included - %d index added to filteredIndices", i)
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}

	if len(m.filteredIndices) == 0 {
		// No matches found prevent index panics
		m.cursor = -1
	}

	m.jumpToHashID(currentRowHash)
	// Load content back into the viewport now its been filtered?
	m.viewport.SetContent(m.renderTable())
}

// endregion

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

func (m *model) statusView() string {
	filterApplied := false
	if m.filterRegex != nil {
		filterApplied = true
	}
	left := fmt.Sprintf("%s • [FILTER: %t] • [MARKS ONLY:%t]", defaultSaveName(*m), filterApplied, m.showOnlyMarked)
	rowsShown, rowsTotal := m.cursor+1, len(m.rows)
	center := fmt.Sprintf("Rows %d/%d", rowsShown, rowsTotal)
	right := "? help • f filter • c comment"

	// Base bar style
	bar := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("238"))

	// Use the bar as the base for every chunk so the background is consistent
	chunk := func() lipgloss.Style { return bar.Copy().Padding(0) }

	// Separator inherits background from bar, only changes the fg
	sep := chunk().Foreground(lipgloss.Color("240")).Render("│")

	total := m.viewport.Width
	if total <= 0 {
		total = 80
	}

	// Compute inner width = total minus bar frame (padding/border)
	hpad, _ := bar.GetFrameSize()
	inner := total - hpad
	if inner < 0 {
		inner = 0
	}

	sepW := lipgloss.Width(sep) // usually 1
	avail := inner - (2 * sepW)
	if avail < 0 {
		avail = 0
	}

	lw := int(float64(avail) * 0.40)
	cw := int(float64(avail) * 0.30)
	rw := avail - lw - cw

	L := chunk().Width(lw).Render(left)
	C := chunk().Width(cw).Align(lipgloss.Center).Render(center)
	R := chunk().Width(rw).Align(lipgloss.Right).Render(right)

	line := lipgloss.JoinHorizontal(lipgloss.Top, L, sep, C, sep, R)
	return bar.Width(total).Render(line)
}

// footerView renders the 2-line footer using local (function-scoped) styles/state.
// width is the terminal width (e.g. m.width from tea.WindowSizeMsg).
func (m *model) footerView(width int) string {
	log.Printf("footerView mode=%d cmd=%d\n", m.currentMode, m.ci.cmd)
	styles := DefaultFooterStyles()

	footerMode := CmdNone
	modeInput := ""
	switch m.currentMode {
	case modeView:
		footerMode = CmdNone
	case modeFilter:
		footerMode = CmdFilter
	case modeMarking:
		footerMode = CmdMark
	case modeComment:
		footerMode = CmdComment
	case modeCommand:
		switch m.ci.cmd {
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

	st := FooterState{
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
	if m.noticeMsg != "" {
		st.StatusMessage = noticeInline(m.noticeMsg, m.noticeType)
	}

	return RenderFooter(width, st, styles)
}

func (m *model) noticeView(msg string, kind string) string {
	if msg == "" {
		return "" // nothing to show
	}

	total := m.viewport.Width
	if total <= 0 {
		total = 80
	}

	// pick an icon & background color based on kind
	var icon, bg string
	switch kind {
	case "info":
		icon, bg = "ℹ", "24" // blue
	case "success":
		icon, bg = "✓", "22" // green
	case "warn":
		icon, bg = "!", "130" // orange
	case "error":
		icon, bg = "×", "160" // red
	default:
		icon, bg = "", "238" // neutral gray
	}

	// build the lineg
	text := msg
	if icon != "" {
		text = fmt.Sprintf("%s %s", icon, msg)
	}

	// style and pad to width
	st := lipgloss.NewStyle().
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color("0")). // black text
		Padding(0, 1)

	// truncate if needed to fit terminal width
	return st.Width(total).Render(text)
}

func noticeInline(msg string, kind string) string {
	if msg == "" {
		return ""
	}
	var icon string
	switch kind {
	case "info":
		icon = "ℹ"
	case "success":
		icon = "✓"
	case "warn":
		icon = "!"
	case "error":
		icon = "×"
	default:
		icon = ""
	}
	if icon == "" {
		return msg
	}
	return fmt.Sprintf("%s %s", icon, msg)
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
	if m.drawerOpen {
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
	markerWidth := len(fmt.Sprintf("%d", len(m.rows))) + utf8.RuneCountInString(commentMarker) // +3 is your padding

	// Standard mark seems to reset any bg colour attempts to need to render anything preceding it
	firstLineMarker := standardMarker + rowBgStyle.Render(fmt.Sprintf("%*d", markerWidth, row.originalIndex))
	additionalLineMarker := standardMarker + rowBgStyle.Render(strings.Repeat(" ", markerWidth))

	if commentPresent {
		firstLineMarker = standardMarker + rowBgStyle.Render(commentMarker+fmt.Sprintf("%*d", markerWidth-utf8.RuneCountInString(commentMarker), row.originalIndex))
		//firstLineMarker = standardMarker + rowBgStyle.Render("[*]")
	}
	//TODO Replace:4 with the width of marker, and comment
	//content := row.Render(cellStyle, m.viewport.Width-4, columnWeights) // Adjust for marker width
	//TODO: May need to think what to do with that 4
	content := row.Render(cellStyle, m.header)
	lines := strings.Split(content, "\n")

	for i := range lines {
		left := additionalLineMarker
		right := rowBgStyle.Render(rowFgStyle.Render(lines[i]))
		if i == 0 { // first line
			left = firstLineMarker
		}
		lines[i] = left + right
		// lines[i] = rowBgStyle.Render(lines[i])
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

func defaultExportName(m model) string {
	if m.lastExportFileName != "" {
		return m.lastExportFileName
	}

	base := strings.TrimSuffix(m.InitialPath, filepath.Ext(m.InitialPath))
	return "export-" + base + ".csv"

}

func defaultSaveName(m model) string {
	// Case 1: we already have a current filename — use it
	if m.fileName != "" {
		return m.fileName
	}

	// Case 2: otherwise fall back to whatever initial path we have
	initial := m.InitialPath
	if initial == "" {
		return "output.json" // final fallback
	}

	// Case 3: if the initial path already ends with .json, use it
	if strings.HasSuffix(strings.ToLower(initial), ".json") {
		return initial
	}

	// Case 4: replace any existing extension with .json
	base := strings.TrimSuffix(initial, filepath.Ext(initial))
	return base + ".json"
}

func (m *model) renderTable() string {
	// TODO: refactor renderTable into view file, and the calculation into a windowAroundCursorFunction in ops

	log.Println("renderTable called")
	viewportHeight := m.viewport.Height
	viewPortWidth := m.viewport.Width
	_ = viewPortWidth // TODO: I'm going to use this just need to remember why and where

	cursor := m.cursor

	if len(m.filteredIndices) == 0 && cursor < 0 {
		log.Printf("renderTable: Returning blank filteredIndices Lenght[%d] cursor[%d]", len(m.filteredIndices), cursor)

		return ""
	}
	//TODO: Defect here as we should be using the row count not the display index to maintain between a filter and non-filtered list
	if len(m.filteredIndices) < cursor {
		m.cursor = 0
		cursor = 0
	}
	var renderedRows []string

	// // Render cursor row first and make sure its 'selected'
	cursorRenderedRow, cursorRenderedRowHeight, _ := m.renderRowAt(cursor)

	if m.drawerOpen {
		m.refreshDrawerContent()
	}

	renderedRows = append([]string{cursorRenderedRow}, renderedRows...)

	heightFree := viewportHeight - cursorRenderedRowHeight

	upIndex := cursor - 1
	downIndex := cursor + 1
	rowCount := 0 // Number of visible rows needed for paging

	// Add rows above and below current cursor until all the free space in the ViewPort is failed (which gives you a calculated page size)
	// Page size needs to be calculate if you have rows that render over 1 line. Best way to achieve word wrapping
	for heightFree > 0 && (upIndex >= 0 || downIndex < len(m.filteredIndices)) {
		if upIndex >= 0 {
			rendered, height, ok := m.renderRowAt(upIndex)
			if ok && height <= heightFree {
				renderedRows = append([]string{rendered}, renderedRows...)
				heightFree -= height
				upIndex--
				rowCount++
				continue
			}
		}
		if downIndex < len(m.filteredIndices) {
			rendered, height, ok := m.renderRowAt(downIndex)
			if ok && height <= heightFree {
				renderedRows = append(renderedRows, rendered)
				heightFree -= height
				downIndex++
				rowCount++
				continue
			}
		}
		break
	}
	m.pageRowSize = rowCount
	m.lastVisibleRowCount = 4
	// Combine rendered rows into a string with proper vertical order
	var b strings.Builder
	for _, r := range renderedRows {
		b.WriteString(r + "\n")
	}

	return b.String()
}
