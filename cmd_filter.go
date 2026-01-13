package main

import (
	"log"
	"regexp"
)

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

// region Filtering

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
