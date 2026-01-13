package main

import "strings"

func (m *model) searchOnce(query string) {
	if query == "" {
		return
	}

	for i, row := range m.rows {
		if strings.Contains(strings.ToLower(row.String()), strings.ToLower(query)) {
			m.cursor = i
			m.viewport.SetYOffset(i)
			return
		}
	}
}
