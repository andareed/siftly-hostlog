package main

import (
	"regexp"

	"github.com/andareed/siftly-hostlog/logging"
)

func (m *model) setFilterPattern(pattern string) error {
	logging.Infof("Setting Pattern to: %s", pattern)
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

// region Filtering

func (m *model) includeRow(row renderedRow) bool {
	if m.showOnlyMarked {
		if _, ok := m.markedRows[row.id]; !ok {
			return false
		}
	}

	if m.filterRegex != nil {
		match := m.filterRegex.MatchString(row.String())
		if !match {
			return false
		}
	}
	return true
}
