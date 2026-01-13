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

	log.Printf("model:Update called with msg: %#T: %#v\n", msg, msg)

	if cmd, handled := m.handleSystemMsg(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleDialogInput(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleWindowMsg(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleDialogMsg(msg); handled {
		return m, cmd
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyMsg(keyMsg)
	}
	return m, nil
}

func (m *model) handleSystemMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch fmt.Sprintf("%T", msg) {
	case "cursor.BlinkMsg", "cursor.BlinkCanceledMsg":
		return nil, true
	}
	switch msg := msg.(type) {
	case clearNoticeMsg:
		if msg.id == m.noticeSeq {
			m.noticeMsg = ""
			m.noticeType = ""
		}
		return nil, true
	}
	return nil, false
}

func (m *model) handleDialogInput(msg tea.Msg) (tea.Cmd, bool) {
	if m.activeDialog == nil || !m.activeDialog.IsVisible() {
		return nil, false
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false
	}
	log.Printf("DIALOG UPDATE: %T got key %q\n", m.activeDialog, km.String())
	log.Printf("model:Update:: Dialog box is active forward update to it\n")
	var cmd tea.Cmd
	m.activeDialog, cmd = m.activeDialog.Update(km)
	return cmd, true
}

func (m *model) handleWindowMsg(msg tea.Msg) (tea.Cmd, bool) {
	win, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return nil, false
	}
	m.terminalHeight = win.Height
	m.terminalWidth = win.Width
	m.viewport = viewport.New(0, 0) // TODO: PRetty sure this is redundant
	m.ready = true
	m.refreshView("window-size", true)
	return nil, true
}

func (m *model) handleDialogMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case dialogs.HelpRequestedMsg:
		log.Printf("Updated was called with msg HelpRequestedMsg (should pop a help dialog box)\n")
		m.activeDialog = dialogs.NewHelpDialog(Keys.Legend())
		m.activeDialog.Show()
		return nil, true
	case dialogs.SaveRequestedMsg:
		log.Printf("Update was called with msg SaveRequestedMsg (should pop a dialog box)\n")
		m.activeDialog = dialogs.NewSaveDialog(defaultSaveName(*m), filepath.Dir(m.fileName))
		m.activeDialog.Show()
		return nil, true
	case dialogs.SaveConfirmedMsg:
		log.Printf("model:Update::Received SaveConfirmedMsg saving to the current file using path %q\n", msg.Path)
		m.activeDialog.Hide()
		if err := SaveModel(m, msg.Path); err != nil {
			return m.startNotice("Error", "", noticeDuration), true
		}
		m.fileName = msg.Path
		return m.startNotice("Saved succeeded", "", noticeDuration), true
	case dialogs.SaveCanceledMsg:
		log.Printf("model:Update::Received SaveCanceledMsg from dialog and hiding the active dialog\n")
		m.activeDialog.Hide()
		return nil, true
	case dialogs.ExportRequestedMsg:
		log.Printf("Update wwas called with msg ExportRequestMsg (pop dialog for exports)\n")
		m.activeDialog = dialogs.NewExportDialog(defaultExportName(*m), filepath.Dir(m.fileName))
		// TODO: What filename should this default to for an export?
		m.activeDialog.Show()
		return nil, true
	case dialogs.ExportConfirmedMsg:
		log.Printf("module:Update::ExportConfirmedMsg begin exporting to the file\n")
		m.activeDialog.Hide()
		if err := ExportModel(m, msg.Path); err != nil {
			return m.startNotice("Export Error", "", noticeDuration), true
		}
		m.lastExportFileName = msg.Path
		return m.startNotice("Exported succeeded", "", noticeDuration), true
	case dialogs.ExportCanceledMsg:
		log.Printf("model:Update:: Received ExportCanceledMsg, close down the dialog\n")
		m.activeDialog.Hide()
		return nil, true
	}
	return nil, false
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

func (m *model) refreshView(reason string, withLayout bool) {
	log.Printf("refreshView: reason=%s layout=%t", reason, withLayout)
	if withLayout {
		m.recomputeLayout(m.terminalHeight, m.terminalWidth)
	}
	if m.drawerOpen {
		m.refreshDrawerContent()
	}
	m.viewport.SetContent(m.renderTable())
}

func (m *model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	didRefresh := false

	switch {
	// Migrating to a command / input method
	case key.Matches(msg, Keys.JumpToLineNo):
		log.Println("Enabling Command: Jumping to specific line number if it exists")
		cmd = m.enterCommand(CmdJump, "", true, false)
	case key.Matches(msg, Keys.Filter):
		log.Println("Enabling Command: Filtering")
		cmd = m.enterCommand(CmdFilter, "", true, false)
	case key.Matches(msg, Keys.MarkMode):
		log.Println("Enable COmmand: Marking")
		cmd = m.enterCommand(CmdMark, "", true, false)
	case key.Matches(msg, Keys.EditComment):
		log.Println("Enable COmmand: Edit Comment")
		cmd = m.enterCommand(CmdComment, "", true, false)
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
		m.refreshView("drawer-toggle", true)
		didRefresh = true
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
	if m.ready && !didRefresh {
		m.refreshView("view-key", false)
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
		st.StatusMessage = noticeText(m.noticeMsg, m.noticeType)
	}

	return RenderFooter(width, st, styles)
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
	renderedRows, rowCount := m.visibleRowsAroundCursor(cursor, viewportHeight)
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

func (m *model) visibleRowsAroundCursor(cursor int, viewportHeight int) ([]string, int) {
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
