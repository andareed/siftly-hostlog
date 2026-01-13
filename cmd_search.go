package main

import "strings"

func (m *model) setSearchQuery(query string) {
	m.ui.searchQuery = strings.TrimSpace(query)
}

func (m *model) searchNext() bool {
	return m.searchFrom(m.cursor+1, 1)
}

func (m *model) searchPrev() bool {
	return m.searchFrom(m.cursor-1, -1)
}

func (m *model) searchFrom(start int, dir int) bool {
	q := strings.TrimSpace(m.ui.searchQuery)
	if q == "" || len(m.data.filteredIndices) == 0 {
		return false
	}

	n := len(m.data.filteredIndices)
	if start < 0 {
		start = n - 1
	}
	if start >= n {
		start = 0
	}

	for i := 0; i < n; i++ {
		idx := start + i*dir
		if idx < 0 {
			idx += n
		}
		if idx >= n {
			idx -= n
		}
		row := m.data.rows[m.data.filteredIndices[idx]]
		if strings.Contains(strings.ToLower(row.String()), strings.ToLower(q)) {
			m.cursor = idx
			return true
		}
	}
	return false
}
