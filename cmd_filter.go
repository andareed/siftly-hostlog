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

func (m *model) includeRow(row renderedRow) bool {
	if m.data.showOnlyMarked {
		if _, ok := m.data.markedRows[row.id]; !ok {
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
