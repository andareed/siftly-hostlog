package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

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

func (m *model) handleMarkCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentMode = modeView
		return m, nil

	case "r", "g", "a", "c":
		var mark MarkColor
		switch msg.String() {
		case "r":
			mark = MarkRed
		case "g":
			mark = MarkGreen
		case "a":
			mark = MarkAmber
		case "c":
			mark = MarkNone
		}

		m.MarkCurrent(mark)
		m.currentMode = modeView

		m.refreshView("mark", false)

		// Notice only on actual change
		return m, m.startNotice(
			fmt.Sprintf("Row %d marked [%s]", m.cursor+1, msg.String()),
			"",
			noticeDuration,
		)
	}

	// Unhandled keys: stay in mark mode, do nothing
	return m, nil
}

func (m *model) jumpToNextMark() {
	log.Println("jumpToNextMark callled..")
	if !m.checkViewPortHasData() {
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
