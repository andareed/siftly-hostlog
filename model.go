package main

import (
	"log"
	"regexp"
	"strings"

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
)

//TODO Lets save all of its as JSON for when working on.
//TODO Lets have a button to clear a marked back to normal shall we
//TODO Replace the name renderedRow, as these are not rendered anymore

type model struct {
	header              renderedRow // single row for column titles in headerview
	rows                []renderedRow
	viewport            viewport.Model
	ready               bool
	cursor              int // index into rows
	lastVisibleRowCount int
	currentMode         mode
	markedRows          map[uint64]MarkColor // map row index to color code
	filterRegex         *regexp.Regexp
	filteredIndices     []int // to store the list of indicides that match the current regex
	filterInput         textinput.Model
	commentInput        textinput.Model
}

func initialModel(data [][]string) *model {
	rows := make([]renderedRow, 0, len(data))
	header := renderedRow{
		cols:   data[0],
		height: 1,
	}

	for _, csvRow := range data[1:] {
		//TODO: Move this to a construct NewRenderedRow in the row.go file
		row := renderedRow{
			cols:   csvRow, // store columns directly
			height: 1,      // assume 1 for now; adjust if multiline logic added later
		}
		row.id = row.ComputeID() // Should be always called therefore should be in the constructor
		rows = append(rows, row)
	}

	fi := textinput.New()
	fi.Placeholder = "Regex Filter..."
	fi.Focus()
	fi.CharLimit = 156
	fi.Width = 50

	return &model{
		header:      header,
		rows:        rows,
		currentMode: modView,
		markedRows:  make(map[uint64]MarkColor),
		filterInput: fi,
	}
}

func (m *model) Init() tea.Cmd {
	m.applyFilter()
	log.Println("Sourcely has been initialised")
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.WindowSizeMsg:
		m.viewport = viewport.New(msg.Width-6, msg.Height-5)
		m.ready = true
		m.viewport.SetContent(m.renderTable())
		return m, nil
	}

	return m, nil
}

func (m *model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentMode {
	case modView:
		return m.handleViewModeKey(msg)
	case modeMarking:
		return m.handleMarkingModeKey(msg)
	case modeFilter:
		return m.handleFilterKey(msg)
	}

	return m, nil
}

func (m *model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	log.Println("handleFilterKey called..")
	switch msg.String() {
	case "enter", "esc":
		log.Println("Enter or ESC Pressed")
		if m.filterInput.Focused() {
			log.Printf("Input boxed classed as focussed so we can apply filter and reset mode back to View wth text : %s", m.filterInput.Value())
			m.setFilterPattern(m.filterInput.Value())
			m.currentMode = modView
			m.commentInput.Blur()
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
	return m, nil

}

func (m *model) handleViewModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "m":
		m.currentMode = modeMarking
		log.Println("Entering Mode: Marking")
	case "f":
		m.currentMode = modeFilter
		m.filterInput.Focus()
		log.Println("Entering Mode: Filter (Focus Box)")
	case "down", "j":
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
	}
	if m.ready {
		m.viewport.SetContent((m.renderTable()))
	}
	return m, nil
}

func (m *model) pageDown() {
	if m.cursor+m.lastVisibleRowCount < len(m.rows) {
		m.cursor += m.lastVisibleRowCount
	} else {
		m.cursor = len(m.rows) - 1
	}
}

func (m *model) pageUp() {
	m.cursor -= m.lastVisibleRowCount
	if m.cursor < 0 {
		m.cursor = 0
	}
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

func (m *model) applyFilter() {
	log.Printf("applyFilter called")
	m.filteredIndices = m.filteredIndices[:0] // reset slice
	if m.filterRegex == nil {
		log.Println("applyFilter: No filter present in textbox so all incides being added to filteredIncidices")
		// Maybe used clamp?
		for i := range m.rows {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	} else {
		for i, row := range m.rows {
			log.Printf("applyFilter: filter applied checking row[%s] against pattern[%s] \n", row.String(), m.filterRegex)
			if m.filterRegex.MatchString(row.String()) {
				log.Printf("Pattern matched - %d index added to filteredIndices", i)
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
		if len(m.filteredIndices) == 0 {
			// No matches found prevent index panics
			m.cursor = -1
		}
	}
	// Load content back into the viewport now its been filtered?
	m.viewport.SetContent(m.renderTable())
}

// endregion

func (m *model) headerView() string {
	return headerStyle.Render(m.header.Render(cellStyle, m.viewport.Width, columnWeights))
	// return headerStyle.Render("Time,Host,Details,Appliance,MAC Address,IPv6 Address,")
}

func (m *model) footerView() string {
	var sb strings.Builder

	switch m.currentMode {
	case modView:
		sb.WriteString("(q)uit | (m)ark | toggle (s)how only marks | (e)xport marks | (w)rite | (c)omment | (f)ilter text | (n/N)ext mark  | Navigations (Up j Down k) ")
	case modeMarking:
		sb.WriteString("Choose a color: (r)ed (g)reen (a)mber (c)lear | esc:cancel")
	case modeFilter:
		sb.WriteString(inputStyle.Render(m.filterInput.View()))
	}

	return sb.String()
}

func (m *model) View() string {
	if !m.ready {
		return "loading..."
	}
	borderedViewPort := tableStyle.Render(m.viewport.View())

	return appstyle.Render(lipgloss.JoinVertical(lipgloss.Left, m.headerView(), borderedViewPort, m.footerView()))
	// return m.viewport.View() + "\n(↑/↓ to scroll, q to quit)"
}

func (m *model) renderRowAt(filteredIdx int) (string, int, bool) {
	if filteredIdx < 0 || filteredIdx >= len(m.filteredIndices) {
		return "", 0, false
	}

	rowIdx := m.filteredIndices[filteredIdx]
	row := m.rows[rowIdx]
	marker := m.getRowMarker(row.id)
	content := row.Render(cellStyle, m.viewport.Width-2, columnWeights) // Adjust for marker width
	lines := strings.Split(content, "\n")

	for i := range lines {
		lines[i] = marker + " " + lines[i] // prepend marker
	}

	rendered := strings.Join(lines, "\n")
	return rendered, row.height, true
}

func (m *model) getRowMarker(index uint64) string {
	switch m.markedRows[index] {
	case MarkRed:
		return redMarker
	case MarkGreen:
		return greenMarker
	case MarkAmber:
		return amberMarker
	default:
		return defaultMarker
	}
}

func (m *model) renderTable() string {
	viewportHeight := m.viewport.Height
	viewPortWidth := m.viewport.Width
	cursor := m.cursor

	if len(m.filteredIndices) == 0 || cursor < 0 || len(m.filteredIndices) <= cursor {
		return ""
	}

	var renderedRows []string

	// // Render cursor row first and make sure its 'selected'
	filteredCursor := m.filteredIndices[cursor]
	cursorRow := &m.rows[filteredCursor]
	currentRenderedRow := selectedStyle.Render(cursorRow.Render(cellStyle, viewPortWidth-2, columnWeights)) // sets cursorRow.height
	marker := m.getRowMarker(cursorRow.id)
	lines := strings.Split(currentRenderedRow, "\n")

	for i := range lines {
		lines[i] = marker + " " + lines[i] // prepend marker
	}

	rendered := strings.Join(lines, "\n")

	renderedRows = append(renderedRows, rendered)

	heightFree := viewportHeight - cursorRow.height

	upIndex := cursor - 1
	downIndex := cursor + 1
	rowCount := 0 // Number of visible rows needed for paging

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

	m.lastVisibleRowCount = 4
	// Combine rendered rows into a string with proper vertical order
	var b strings.Builder
	for _, r := range renderedRows {
		b.WriteString(r + "\n")
	}

	return b.String()
}
