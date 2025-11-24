package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
	modView mode = iota
	modeFilter
	modeMarking
	modeComent
)

//TODO Lets save all of its as JSON for when working on.
//TODO Lets have a button to clear a marked back to normal shall we
//TODO Replace the name renderedRow, as these are not rendered anymore

type model struct {
	header              renderedRow // single row for column titles in headerview
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
	// Might be messy but all tick related messages need to be handle first and discard if necessary
	switch fmt.Sprintf("%T", msg) {
	case "cursor.BlinkMsg", "cursor.BlinkCanceledMsg":
		return m, nil // Swallow the sound of the constant Blink Msgs
	}
	log.Printf("model:Update called with msg: %#T: %#v\n", msg, msg)

	if m.activeDialog != nil && m.activeDialog.IsVisible() {
		switch msg.(type) {
		case tea.KeyMsg: //May need to WindowSizeMsg etc.. at a later date
			log.Printf("model:Update:: Dialog box is active forward update to it")
			// TODO Probably a good idea to write out a log with the type thats current active to save debug frustration
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
		log.Printf("Updated was called with msg HelpRequestedMsg (should pop a help dialog box)")
		m.activeDialog = dialogs.NewHelpDialog(Keys.Legend())
		m.activeDialog.Show()
	case dialogs.SaveRequestedMsg:
		log.Printf("Update was called with msg SaveRequestedMsg (should pop a dialog box)")
		m.activeDialog = dialogs.NewSaveDialog(defaultSaveName(*m), filepath.Dir(m.fileName))
		m.activeDialog.Show()
	case dialogs.SaveConfirmedMsg:
		log.Printf("model:Update::Received SaveConfirmedMsg saving to the current file using path %q\n", msg.Path)
		m.activeDialog.Hide()
		if err := SaveModel(m, msg.Path); err != nil {
			cmd := m.startNotice("Error", "", 1500*time.Millisecond)
			return m, cmd
		}
		cmd := m.startNotice("Saved succeeded", "", 1500*time.Millisecond)
		m.fileName = msg.Path
		return m, cmd
	case dialogs.SaveCanceledMsg:
		log.Printf("model:Update::Received SaveCanceledMsg from dialog and hiding the active dialog\n")
		m.activeDialog.Hide()
	case dialogs.ExportRequestedMsg:
		log.Printf("Update wwas called with msg ExportRequestMsg (pop dialog for exports)")
		m.activeDialog = dialogs.NewExportDialog(defaultExportName(*m), filepath.Dir(m.fileName)) // TODO: What filename should this default to for an export?
		m.activeDialog.Show()
	case dialogs.ExportConfirmedMsg:
		// Screen Select should be exported as a csv
		log.Printf("module:Update::ExportConfirmedMsg begin exporting to the file")
		m.activeDialog.Hide()
		// TODO: Insert call for exporting current selection to the csv
		if err := ExportModel(m, msg.Path); err != nil {
			cmd := m.startNotice("Export Error", "", 1500*time.Millisecond)
			return m, cmd
		}
		cmd := m.startNotice("Exported succeeded", "", 1500*time.Millisecond)
		m.lastExportFileName = msg.Path
		return m, cmd
	case dialogs.ExportCanceledMsg:
		log.Printf("model:Update:: Received ExportCanceledMsg, close down the dialog")
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
		height -= m.drawerHeight
		m.drawerPort.Width = width // Minus out the padding.
		m.commentInput.SetWidth(width)
		m.commentInput.SetHeight(8)
		m.drawerPort.Height = 8
		m.drawerPort.Width = width
	}
	log.Printf("Update Received of type Windows Size Message. ViewPort was [%d] and is now getting set to height[%d] width [%d]/n", m.viewport.Height, height, width)
	m.viewport.Height = height
	m.viewport.Width = width
}

func (m *model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// if m.activeDialog != nil && m.activeDialog.IsVisible() {
	//
	// return m.handleDialogKey(msg)
	// }
	switch m.currentMode {
	case modView:
		return m.handleViewModeKey(msg)
	case modeMarking:
		return m.handleMarkingModeKey(msg)
	case modeFilter:
		return m.handleFilterKey(msg)
	case modeComent:
		return m.handleCommentKey(msg)
	}

	return m, nil
}

// func (m *model) handleDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
// var cmd tea.Cmd
// switch msg.String() {
// case "enter", "esc":
// log.Printf("handleDialogKey: Enter or esc presssed on dialog")
// }
// return m, cmd
//  }

func (m *model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	log.Println("handleFilterKey called..")
	switch msg.String() {
	case "enter", "esc":
		log.Println("Enter or ESC Pressed")
		//TODO: Defect here, when we edit twice this is no longer focussed. Why?
		if m.filterInput.Focused() {
			log.Printf("Input boxed classed as focussed so we can apply filter and reset mode back to View wth text : %s", m.filterInput.Value())
			m.setFilterPattern(m.filterInput.Value())
			m.currentMode = modView
			m.filterInput.Blur()
			//m.applyFilter()
			cmd = m.startNotice(fmt.Sprintf("Filter pattern set to {%s}", m.filterRegex.String()), "", 1500*time.Millisecond)
		}
		return m, cmd
	default:
		log.Println("Generic character received and adding to the filter Input")
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}
	//return m, nil
}

func (m *model) handleCommentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	log.Println("handleCommentKey called..")
	switch msg.String() {
	case "enter", "esc":
		log.Println("Enter or Esc pressed")
		if m.commentInput.Focused() {
			// TODO: If the viewport is being used , will the input still be focussed, think so?
			m.CommentCurrent(m.commentInput.Value()) // Save the comment to the map
			m.currentMode = modView
			m.commentInput.Blur()
			m.viewport.SetContent(m.renderTable())
			cmd = m.startNotice(fmt.Sprintf("Comment {$s} made to row {%d}", m.commentInput.Value(), m.cursor+1), "", 1500*time.Millisecond)
		}
		return m, cmd
	default:
		log.Println("Generic character received and adding to the filter Input")
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	}

}

func (m *model) MarkCurrent(colour MarkColor) {
	if (m.cursor) < 0 || m.cursor >= len(m.filteredIndices) {
		return // This messed up as the cursor isn't at a point in the viewport
	}
	master := m.filteredIndices[m.cursor] // Gets the row
	id := m.rows[master].id
	if colour == MarkNone {
		delete(m.markedRows, id)
		log.Printf("Cursor: %d with Stable ID %d has been unmarked", m.cursor, id)
	} else {
		log.Printf("Cursor: %d with Stable ID %d is being marked with color %s", m.cursor, id, colour)
		m.markedRows[id] = colour
	}
}

func (m *model) handleMarkingModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	log.Println("handleMarkingModeKey called..")
	//TODO: We can axe this switch now in favour or just using the enum
	switch msg.String() {
	case "r":
		m.MarkCurrent(MarkRed)
		m.currentMode = modView
	case "g":
		m.MarkCurrent(MarkGreen)
		m.currentMode = modView
	case "a":
		m.MarkCurrent(MarkAmber)
		m.currentMode = modView
	case "c":
		m.MarkCurrent(MarkNone)
		m.currentMode = modView
	case "esc":
		m.currentMode = modView
	}
	cmd = m.startNotice(fmt.Sprintf("Row {%d} marked with colour {%s}", m.cursor+1, msg.String()), "", 1500*time.Millisecond)
	m.viewport.SetContent(m.renderTable()) //TODO: Should the Setcontent and Renders be part of a proper update call. This is just ha hack (same as marking with a comment)
	return m, cmd

}

func (m *model) handleViewModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, Keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, Keys.CopyRow):
		log.Println("Key Combination for CopyRow To Clipboard")
		cmd = m.copyRowToClipboard()
	case key.Matches(msg, Keys.MarkMode):
		m.currentMode = modeMarking
		log.Println("Entering Mode: Marking")
	case key.Matches(msg, Keys.ShowMarksOnly):
		// Show Marks only
		log.Println("Toggle for Show Marks Only has been pressed")
		m.showOnlyMarked = !m.showOnlyMarked
		cmd = m.startNotice(fmt.Sprintf("'Show Only Marked Rows' toggled {%b}", m.showOnlyMarked), "", 1500*time.Millisecond)
		m.applyFilter()
	case key.Matches(msg, Keys.NextMark):
		// Next mark jump
		log.Println("Here we go; jumping to the next mark")
		m.jumpToNextMark()
	case key.Matches(msg, Keys.PrevMark):
		log.Println("Back once again: jumping to the previous mark")
		m.jumpToPreviousMark()
		m.ready = true
	case key.Matches(msg, Keys.Filter):
		// Set Filter
		m.currentMode = modeFilter
		m.filterInput.Focus()
		log.Println("Entering Mode: Filter (Focus Box)")
	case key.Matches(msg, Keys.ClearFilter):
		// Clear Filter
		log.Println("Shift F, clearing Filter")
		m.setFilterPattern("") // Set the filter to nothing which will clear
		cmd = m.startNotice("Cleared filter", "", 1500*time.Millisecond)
	case key.Matches(msg, Keys.EditComment):
		// Comment (Edit) if the drawer is open (i.e. C has been pressed previously)
		if m.drawerOpen {
			m.commentInput.Focus()
			m.currentMode = modeComent
		}
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
			return m.startNotice("Error with Clipboard occurred.", "", 1500*time.Millisecond)
		} else {
			return m.startNotice("Copied Row COntent to Clipboard", "", 1500*time.Millisecond)
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

func (m *model) getCommentContent(rowIdx uint64) string {
	// Probably want some error checking around the rowIdx
	if c, ok := m.commentRows[rowIdx]; ok && c != "" {
		return c
	}
	return "" // No comment, so returning blank
}

func (m *model) refreshDrawerContent() {
	log.Printf("refreshDrawerContent called..")
	currentComment := m.getCommentContent(m.currentRowHashID())
	log.Printf("Comment Input and Drawer Port being set to: %s", currentComment)
	m.commentInput.SetValue(currentComment)
	m.drawerPort.SetContent(currentComment)
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

func (m *model) jumpToNextMark() {
	log.Println("jumpToNextMark callled..")
	n := len(m.filteredIndices)
	if n == 0 {
		log.Println("filterIndicies is empty")
		return
	}
	if m.cursor < 0 {
		log.Println("Cursor at 0 or below")
		return
	}

	for i := m.cursor + 1; i < len(m.filteredIndices); i++ {
		rowIdx := m.filteredIndices[i]
		row := m.rows[rowIdx]
		if _, ok := m.markedRows[row.id]; ok {
			log.Printf("Next mark found at %d \n", i)
			m.cursor = i
			return
		}

	}
	log.Println("No next mark has been found")
}

func (m *model) jumpToPreviousMark() {
	log.Println("jumpToPreviousMark called..")
	n := len(m.filteredIndices)
	if n == 0 {
		log.Println("filteredIndicies is emtpy")
	}
	if m.cursor < 0 {
		log.Println("Cursor at 0 or below")
	}

	for i := m.cursor - 1; i >= 0; i-- {
		rowIdx := m.filteredIndices[i]
		row := m.rows[rowIdx]
		if _, ok := m.markedRows[row.id]; ok {
			log.Println("Previous mark has been found")
			m.cursor = i
			return
		}

	}
	log.Println("No previous mark has been found")
}

func (m *model) CommentCurrent(comment string) {
	log.Printf("CommentCurrent called..\n")
	if (m.cursor) < 0 || m.cursor >= len(m.filteredIndices) {
		return
	}

	idx := m.filteredIndices[m.cursor]
	hashId := m.rows[idx].id
	if comment == "" {
		delete(m.commentRows, hashId)
		log.Printf("Clear comment Index[%d] on HashID[%d]\n", idx, hashId)
		return
		//TODO: Probably need this sending a notificatoin
	}
	m.commentRows[hashId] = comment
	log.Printf("Setting Comment[%s] to Index[%d] on HashID[%d]\n", comment, idx, hashId)
}

// region Filtering

func (m *model) setFilterPattern(pattern string) error {
	log.Printf("Setting Pattern to: %s\n", pattern)
	if pattern == "" {
		m.filterRegex = nil
	} else {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		m.filterRegex = re
	}
	m.applyFilter()
	return nil
}

func (m *model) includeRow(row renderedRow) bool {
	log.Printf("includeRow called")

	if m.showOnlyMarked {
		if _, ok := m.markedRows[row.id]; !ok {
			log.Printf("row[%d]: EXCLUDE (not marked)", row.id)
			return false
		}
	}

	if m.filterRegex != nil {
		match := m.filterRegex.MatchString(row.String())
		log.Printf("applyFilter: filter applied checking row[%s] against pattern[%s] \n", row.String(), m.filterRegex)
		if !match {
			return false
		}
	}
	log.Printf("applyFilter: %s is to be included", row.String())
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
	// Max width of RowNumbers, plus the size of the pillmarker,a nd comment marker
	markerWidth := len(fmt.Sprintf("%d", len(m.rows))) + utf8.RuneCountInString(pillMarker) + utf8.RuneCountInString(commentMarker) // +3 is the padding for whether a comment [*]
	return headerStyle.Render(strings.Repeat(" ", markerWidth) + m.header.Render(cellStyle, m.viewport.Width-4, columnWeights))
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

	// build the line
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

func (m *model) footerView() string {
	// Deprecating for statusView
	var sb strings.Builder

	switch m.currentMode {
	case modeMarking:
		sb.WriteString(m.noticeView("Choose a color: (r)ed (g)reen (a)mber (c)lear | esc:cancel", ""))
	case modeFilter:
		sb.WriteString(inputStyle.Render(m.filterInput.View()))
	case modeComent:
		sb.WriteString("enter:save | esc:cancel")
		//TODO:Not sure if i need this for drawerPort yet?
		//sb.WriteString(m.drawerPort.View())
		//sb.WriteString(inputStyle.Render(m.commentInput.View()))
	default:
		sb.WriteString(m.noticeView(m.noticeMsg, m.noticeType))
	}
	sb.WriteString("\n")
	sb.WriteString(m.statusView())
	return sb.String()
}

func (m *model) View() string {
	if !m.ready {
		return "loading..."
	}

	// 1) Build your base content (with or without drawer)
	bordered := tableStyle.Render(m.viewport.View())

	var main string
	if m.drawerOpen {
		drawerTitle := headerStyle.Render("Comments\n")
		switch m.currentMode {
		case modView:
			// no side effects here (no Blur/SetCursor)
			drawer := tableStyle.Render(drawerTitle + "\n" + m.commentInput.View())
			main = appstyle.Render(lipgloss.JoinVertical(
				lipgloss.Left, m.headerView(), bordered, drawer, m.footerView(),
			))
		case modeComent:
			drawer := tableStyle.Render(drawerTitle + "\n" + m.commentInput.View())
			main = appstyle.Render(lipgloss.JoinVertical(
				lipgloss.Left, m.headerView(), bordered, drawer, m.footerView(),
			))
		}
	} else {
		main = appstyle.Render(lipgloss.JoinVertical(
			lipgloss.Left, m.headerView(), bordered, m.footerView(),
		))
	}

	// 2) If a dialog is showing, draw the overlay + dialog instead
	// (simple/robust approach; replaces base while modal is active)
	if m.activeDialog != nil && m.activeDialog.IsVisible() {
		// Prefer real dimensions you track (e.g., from WindowSizeMsg)
		w, h := m.viewport.Width, m.viewport.Height
		return lipgloss.Place(
			w, h, // from WindowSizeMsg
			lipgloss.Center, lipgloss.Center,
			m.activeDialog.View(), // modal box only (no centering here)
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("236")), // solid black backdrop
		)

	}

	// 3) Otherwise, return the normal UI
	return main
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
	content := row.Render(cellStyle, m.viewport.Width-4, columnWeights) // Adjust for marker width

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
