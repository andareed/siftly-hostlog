package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

// func initialModel(data [][]string) *model {
// 	// Is passed the CSV as an array of strings and initialises the model as a set of rows.
// 	rows := make([]renderedRow, 0, len(data))
// 	header := renderedRow{
// 		// TODO: How to add additional columns cleanly
// 		// cols:   append([]string{"Comment"}, data[0]...),
// 		cols:          data[0],
// 		height:        1,
// 		originalIndex: 0,
// 	}

// 	for i, csvRow := range data[1:] {
// 		//TODO: Move this to a construct NewRenderedRow in the row.go file
// 		row := renderedRow{
// 			cols:   csvRow, // store columns directly
// 			height: 1,      // assume 1 for now; adjust if multiline logic added later
// 		}
// 		row.id = row.ComputeID() // Should be always called therefore should be in the constructor
// 		row.originalIndex = i + 1
// 		rows = append(rows, row)

// 	}

// 	fi := textinput.New()
// 	fi.Placeholder = "Regex Filter..."
// 	fi.Focus()
// 	fi.CharLimit = 156
// 	fi.Width = 50

// 	ca := textarea.New()
// 	ca.Placeholder = "Comment:"
// 	ca.Focus()
// 	ca.CharLimit = 256
// 	//ca.Width = 150

// 	return &model{
// 		header:         header,
// 		rows:           rows,
// 		currentMode:    modView,
// 		markedRows:     make(map[uint64]MarkColor),
// 		commentRows:    make(map[uint64]string),
// 		filterInput:    fi,
// 		commentInput:   ca,
// 		showOnlyMarked: false,
// 		drawerPort:     viewport.New(0, 0),
// 		drawerHeight:   13,
// 		drawerOpen:     false,
// 	}
// }

func (m *model) Init() tea.Cmd {
	m.applyFilter()
	log.Println("siftly-hostlog: Initialised")
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.WindowSizeMsg:

		// Set Drawer Width to match the above to keep them in line
		// Height gets set when C is pressed so should not needed.
		//m.drawerPort.Width = msg.Width - 6
		//if m.drawerOpen {
		//m.viewport.Height = msg.Height - 5 - m.drawerHeight
		//}
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width
		m.viewport = viewport.New(0, 0) // TODO: PRetty sure this is redundant
		m.recomputeLayout(m.terminalHeight, m.terminalWidth)
		m.ready = true
		m.viewport.SetContent(m.renderTable())

		return m, nil
	}

	return m, nil
}

func (m *model) recomputeLayout(height int, width int) {
	// Computes the layout based on whats being rendered
	log.Printf("recomputeLayout called with height[%d] width[%d]\n", height, width)
	height -= 5
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
	m.viewport.SetContent(m.renderTable()) //TODO: Should the Setcontent and Renders be part of a proper update call. This is just ha hack (same as marking with a comment)
	return m, nil

}

func (m *model) handleViewModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "m":
		m.currentMode = modeMarking
		log.Println("Entering Mode: Marking")
	case "M":
		// Show Marks only
		log.Println("Toggle for Show Marks Only has been pressed")
		m.showOnlyMarked = !m.showOnlyMarked
		m.applyFilter()
	case "n":
		// Next mark jump
		log.Println("Here we go; jumping to the next mark")
		m.jumpToNextMark()
	case "shift+n", "N":
		log.Println("Back once again: jumping to the previous mark")
		m.jumpToPreviousMark()
	case "f":
		m.currentMode = modeFilter
		m.filterInput.Focus()
		log.Println("Entering Mode: Filter (Focus Box)")
	case "shift+f", "F":
		log.Println("Shift F, clearing Filter")
		m.setFilterPattern("") // Set the filter to nothing which will clear
	case "e":
		if m.drawerOpen {
			m.commentInput.Focus()
			m.currentMode = modeComent
		}
	case "c":
		//m.currentMode = modeComent
		//m.commentInput.Focus()
		//m.loadOrClearCommentBox()
		m.drawerOpen = !m.drawerOpen
		log.Printf("handleViewModeKey: Toggling Drawer (bottom view above the footer) now see to [%t]", m.drawerOpen)
		m.recomputeLayout(m.terminalHeight, m.terminalWidth)
	case "down", "j":
		log.Printf("handleViewModekey: Down or J pressed, moving cursor one position. Cursor [%d] Rows_Total [%d] DrawerOpen[%t]", m.cursor, len(m.rows), m.drawerOpen)
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "u":
		m.pageUp()
	case "d":
		m.pageDown()

	case "left", "h":
		m.viewport.ScrollLeft(4) // tune step
	case "right", "l":
		m.viewport.ScrollRight(4)
	case "w":
		log.Printf("handleViewModeKey: key_press[w] calling SaveMeta with filename")
		if err := SaveModel(m, "snapshot.json"); err != nil { /* handle */
		}

	}

	//TODO: DON'T THINK WE SHOULD BE RENDERING TABLE EVERY TIME TBH
	if m.ready {
		m.viewport.SetContent((m.renderTable()))
	}
	return m, nil
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
	rowIdx := m.filteredIndices[m.cursor]
	hashId := m.rows[rowIdx].id
	return hashId
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
	m.filteredIndices = m.filteredIndices[:0] // reset slice

	if m.filterRegex == nil && !m.showOnlyMarked {
		log.Println("applyFilter: No filter text and showOnly marked is false there all indices being added to filteredIncidices")
		// Maybe used clamp?
		for i := range m.rows {
			m.filteredIndices = append(m.filteredIndices, i)
		}
		if len(m.filteredIndices) == 0 {
			m.cursor = 1
		}
		m.viewport.SetContent(m.renderTable())
		return
	}

	for i, row := range m.rows {
		log.Printf("applyFilter: filter applied checking row[%s] against pattern[%s] \n", row.String(), m.filterRegex)
		if m.includeRow(row) {
			log.Printf("Row included - %d index added to filteredIndices", i)
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}
	if len(m.filteredIndices) == 0 {
		// No matches found prevent index panics
		m.cursor = -1
	}
	// Load content back into the viewport now its been filtered?
	m.viewport.SetContent(m.renderTable())
}

// endregion

func (m *model) headerView() string {
	// Max width of RowNumbers, plus the size of the pillmarker,a nd comment marker
	markerWidth := len(fmt.Sprintf("%d", len(m.rows))) + utf8.RuneCountInString(pillMarker) + utf8.RuneCountInString(commentMarker) // +3 is the padding for whether a comment [*]
	return headerStyle.Render(strings.Repeat(" ", markerWidth) + m.header.Render(cellStyle, m.viewport.Width-4, columnWeights))
}

func (m *model) footerView() string {
	var sb strings.Builder

	switch m.currentMode {
	case modView:
		if m.drawerOpen {
			sb.WriteString("(q)uit  (↑/↓ j/k)nav  (/)filter  (m)mark  (M)marks-only  (n/N)next/prev-mark  (c)comment (e)edit-comment (x)export  (w)write")
		} else {
			sb.WriteString("(q)uit  (↑/↓ j/k)nav  (/)filter  (m)mark  (M)marks-only  (n/N)next/prev-mark  (c)comment  (x)export  (w)write")
		}
	case modeMarking:
		sb.WriteString("Choose a color: (r)ed (g)reen (a)mber (c)lear | esc:cancel")
	case modeFilter:
		sb.WriteString(inputStyle.Render(m.filterInput.View()))
	case modeComent:
		sb.WriteString("enter:save | esc:cancel")
		//TODO:Not sure if i need this for drawerPort yet?
		//sb.WriteString(m.drawerPort.View())
		//sb.WriteString(inputStyle.Render(m.commentInput.View()))
	}

	return sb.String()
}

func (m *model) View() string {
	if !m.ready {
		return "loading..."
	}
	borderedViewPort := tableStyle.Render(m.viewport.View())

	if m.drawerOpen {
		drawerTitle := headerStyle.Render("Comments\n")
		switch m.currentMode {
		case modView:
			log.Printf("View should render comment in viewport with Height [%d], Width [%d]", m.drawerPort.Height, m.drawerPort.Width)
			//drawer := tableStyle.Render(
			//drawerTitle + commentArea.Render(m.drawerPort.View()))
			drawer := tableStyle.Render(
				drawerTitle + "\n" + m.commentInput.View())
			m.commentInput.Blur()
			m.commentInput.SetCursor(0)
			return appstyle.Render(lipgloss.JoinVertical(lipgloss.Left, m.headerView(), borderedViewPort, drawer, m.footerView()))
		case modeComent:
			drawer := tableStyle.Render(
				drawerTitle + "\n" + m.commentInput.View())
			return appstyle.Render(lipgloss.JoinVertical(lipgloss.Left, m.headerView(), borderedViewPort, drawer, m.footerView()))
		}
	}
	return appstyle.Render(lipgloss.JoinVertical(lipgloss.Left, m.headerView(), borderedViewPort, m.footerView()))
	// return m.viewport.View() + "\n(↑/↓ to scroll, q to quit)"
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

func (m *model) renderTable() string {
	log.Println("renderTable called")
	viewportHeight := m.viewport.Height
	viewPortWidth := m.viewport.Width
	_ = viewPortWidth // TODO: I'm going to use this just need to remember why and where

	cursor := m.cursor

	if len(m.filteredIndices) == 0 || cursor < 0 {
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
