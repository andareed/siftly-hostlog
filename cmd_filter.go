package main

import (
	"regexp"

	"github.com/andareed/siftly-hostlog/logging"
)

func (m *model) setFilterPattern(pattern string) error {
	logging.Infof("Setting Pattern to: %s", pattern)
	if pattern == "" {
		m.data.filterRegex = nil
	} else {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		m.data.filterRegex = re
	}
	m.applyFilter()
	return nil
}

// region Filtering

func (m *model) includeRow(row renderedRow, rowIndex int) bool {
	if m.data.showOnlyMarked {
		if _, ok := m.data.markedRows[row.id]; !ok {
			return false
		}
	}

	if m.data.timeWindow.Enabled {
		if rowIndex < 0 || rowIndex >= len(m.data.rowHasTimes) {
			return false
		}
		if !m.data.rowHasTimes[rowIndex] {
			return false
		}
		ts := m.data.rowTimes[rowIndex]
		if ts.Before(m.data.timeWindow.Start) || ts.After(m.data.timeWindow.End) {
			return false
		}
	}

	if m.data.filterRegex != nil {
		match := m.data.filterRegex.MatchString(row.String())
		if !match {
			return false
		}
	}
	return true
}
