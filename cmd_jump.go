package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) jumpToStart() {
	log.Println("jumpToStart called...")
	if !m.checkViewPortHasData() {
		return
	}
	m.cursor = 0
}

func (m *model) jumpToEnd() {
	log.Println("jumpToEnd called...")
	if !m.checkViewPortHasData() {
		return
	}
	// As filteredIndices gets fully populated with all rows if there is no
	// filter. We are safe to say this is the last one, regardless.
	m.cursor = len(m.filteredIndices) - 1
}

func (m *model) jumpToLine(lineNo int) tea.Cmd {
	log.Println("jumpToLineNo")
	if !m.checkViewPortHasData() {
		return nil
	}
	if lineNo <= 0 {
		return m.startNotice(fmt.Sprintf("Line %d out of bounds", lineNo), "warn", noticeDuration)
	}
	target := lineNo - 1
	if target >= len(m.rows) {
		return m.startNotice(fmt.Sprintf("Line %d out of bounds", lineNo), "warn", noticeDuration)
	}
	for i, idx := range m.filteredIndices {
		if idx == target {
			m.cursor = i
			return nil
		}
	}
	return m.startNotice(fmt.Sprintf("Line %d not in current filter", lineNo), "warn", noticeDuration)
}
