package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andareed/siftly-hostlog/logging"
)

func (m *model) markCurrent(colour MarkColor) {
	if (m.cursor) < 0 || m.cursor >= len(m.filteredIndices) {
		return // This messed up as the cursor isn't at a point in the viewport
	}
	master := m.filteredIndices[m.cursor] // Gets the row
	id := m.rows[master].id
	if colour == MarkNone {
		delete(m.markedRows, id)
		logging.Infof("Cursor: %d with Stable ID %d has been unmarked", m.cursor, id)
	} else {
		logging.Infof("Cursor: %d with Stable ID %d is being marked with color %s", m.cursor, id, colour)
		m.markedRows[id] = colour
	}
}

func (m *model) handleMarkCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.ui.mode = modeView
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

		m.markCurrent(mark)
		m.ui.mode = modeView

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
	logging.Debug("jumpToNextMark callled..")
	if !m.checkViewPortHasData() {
		return
	}

	for i := m.cursor + 1; i < len(m.filteredIndices); i++ {
		rowIdx := m.filteredIndices[i]
		row := m.rows[rowIdx]
		if _, ok := m.markedRows[row.id]; ok {
			logging.Debugf("Next mark found at %d", i)
			m.cursor = i
			return
		}

	}
	logging.Debug("No next mark has been found")
}

func (m *model) jumpToPreviousMark() {
	logging.Debug("jumpToPreviousMark called..")
	n := len(m.filteredIndices)
	if n == 0 {
		logging.Debug("filteredIndicies is emtpy")
	}
	if m.cursor < 0 {
		logging.Debug("Cursor at 0 or below")
	}

	for i := m.cursor - 1; i >= 0; i-- {
		rowIdx := m.filteredIndices[i]
		row := m.rows[rowIdx]
		if _, ok := m.markedRows[row.id]; ok {
			logging.Debug("Previous mark has been found")
			m.cursor = i
			return
		}

	}
	logging.Debug("No previous mark has been found")
}
