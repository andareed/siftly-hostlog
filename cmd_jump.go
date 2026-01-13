package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andareed/siftly-hostlog/logging"
)

func (m *model) jumpToStart() {
	logging.Debug("jumpToStart called...")
	if !m.checkViewPortHasData() {
		return
	}
	m.cursor = 0
}

func (m *model) jumpToEnd() {
	logging.Debug("jumpToEnd called...")
	if !m.checkViewPortHasData() {
		return
	}
	// As filteredIndices gets fully populated with all rows if there is no
	// filter. We are safe to say this is the last one, regardless.
	m.cursor = len(m.data.filteredIndices) - 1
}

func (m *model) jumpToLine(lineNo int) tea.Cmd {
	logging.Debug("jumpToLineNo")
	if !m.checkViewPortHasData() {
		return nil
	}
	if lineNo <= 0 {
		return m.startNotice(fmt.Sprintf("Line %d out of bounds", lineNo), "warn", noticeDuration)
	}
	target := lineNo - 1
	if target >= len(m.data.rows) {
		return m.startNotice(fmt.Sprintf("Line %d out of bounds", lineNo), "warn", noticeDuration)
	}
	for i, idx := range m.data.filteredIndices {
		if idx == target {
			m.cursor = i
			return nil
		}
	}
	return m.startNotice(fmt.Sprintf("Line %d not in current filter", lineNo), "warn", noticeDuration)
}
