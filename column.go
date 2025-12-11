package main

import "strings"

type ColumnRole int

const (
	RoleNormal  ColumnRole = iota
	RolePrimary            // Details
	RoleSecondary
)

type ColumnMeta struct {
	Name     string
	Index    int
	Role     ColumnRole
	Visible  bool
	MinWidth int
	Weight   float64
	Width    int
}

func detectRole(name string) ColumnRole {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "details":
		return RolePrimary
	case "id", "time":
		return RoleSecondary
	default:
		return RoleNormal
	}
}

func defaultMinWidthForRole(r ColumnRole) int {
	switch r {
	case RolePrimary:
		return 30
	case RoleSecondary:
		return 12
	default:
		return 8
	}
}

func defaultWeightForRole(r ColumnRole) float64 {
	switch r {
	case RolePrimary:
		return 5.0
	case RoleSecondary:
		return 2.0
	default:
		return 1.0
	}
}

// var columnWeights = []float64{
// 0.1,  // Time
// 0.15, // Host
// 0.4,  // Details
// 0.1,  // Appliance
// 0.15, // MAC
// 0.1,  // IPvc/ c/
// 0,
// }

func markEmptyColumns(cols []ColumnMeta, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	for i := range cols {
		hasData := false
		for _, row := range rows[1:] { // skip header
			if cols[i].Index >= len(row) {
				continue
			}
			if strings.TrimSpace(row[cols[i].Index]) != "" {
				hasData = true
				break
			}
		}
		if !hasData {
			// Column is empty in all data rows → hide it
			// unless you want to keep a primary/details column visible regardless.
			if cols[i].Role != RolePrimary {
				cols[i].Visible = false
				cols[i].Width = 0
				cols[i].Weight = 0
			}
		}
	}
}

func layoutColumns(cols []ColumnMeta, totalWidth int) []ColumnMeta {
	// totalWidth == rowWidth from your TUI (minus borders/padding if needed)
	if totalWidth <= 0 {
		return cols
	}

	// 1. Sum min widths & weights for visible columns
	minSum := 0
	weightSum := 0.0

	for i := range cols {
		if !cols[i].Visible {
			continue
		}
		minSum += cols[i].MinWidth
		weightSum += cols[i].Weight
	}

	if minSum >= totalWidth {
		// Too tight: just give each visible column its MinWidth clamped
		for i := range cols {
			if !cols[i].Visible {
				continue
			}
			if cols[i].MinWidth > totalWidth {
				cols[i].Width = totalWidth // all we can do
			} else {
				cols[i].Width = cols[i].MinWidth
			}
		}
		return cols
	}

	remaining := totalWidth - minSum

	// 2. Distribute remaining space by weight
	for i := range cols {
		if !cols[i].Visible {
			cols[i].Width = 0
			continue
		}

		extra := 0
		if weightSum > 0 {
			extra = int(float64(remaining) * (cols[i].Weight / weightSum))
		}
		cols[i].Width = cols[i].MinWidth + extra
	}

	return cols
}
