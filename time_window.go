package main

import (
	"strings"
	"time"
)

const (
	timeInputLayout = "2006-01-02 15:04:05"
	logTimeLayout   = "Mon Jan 02 15:04:05 MST 2006"
)

type timeWindowResetBehavior int

const (
	timeWindowResetToDefault timeWindowResetBehavior = iota
	timeWindowResetDisable
)

const timeWindowResetMode = timeWindowResetDisable

func (m *model) computeTimeBounds() {
	m.data.timeColumnIndex = findTimeColumnIndex(m.data.header)
	m.data.rowTimes = make([]time.Time, len(m.data.rows))
	m.data.rowHasTimes = make([]bool, len(m.data.rows))

	if m.data.timeColumnIndex < 0 {
		m.data.hasTimeBounds = false
		return
	}

	hasAny := false
	var minTime time.Time
	var maxTime time.Time

	for i, row := range m.data.rows {
		if m.data.timeColumnIndex >= len(row.cols) {
			continue
		}
		raw := row.cols[m.data.timeColumnIndex]
		ts, ok := parseLogTimestamp(raw)
		if !ok {
			continue
		}
		m.data.rowTimes[i] = ts
		m.data.rowHasTimes[i] = true
		if !hasAny {
			minTime = ts
			maxTime = ts
			hasAny = true
			continue
		}
		if ts.Before(minTime) {
			minTime = ts
		}
		if ts.After(maxTime) {
			maxTime = ts
		}
	}

	m.data.hasTimeBounds = hasAny
	if hasAny {
		m.data.timeMin = minTime
		m.data.timeMax = maxTime
	}
}

func findTimeColumnIndex(cols []ColumnMeta) int {
	for i := range cols {
		name := strings.TrimSpace(cols[i].Name)
		name = strings.TrimPrefix(name, "\ufeff")
		if strings.EqualFold(name, "time") {
			return i
		}
	}
	return -1
}

func parseLogTimestamp(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if idx := strings.LastIndex(raw, ":"); idx != -1 {
		raw = strings.TrimSpace(raw[:idx])
	}
	ts, err := time.Parse(logTimeLayout, raw)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func clampTimeToBounds(t time.Time, min time.Time, max time.Time) time.Time {
	if t.Before(min) {
		return min
	}
	if t.After(max) {
		return max
	}
	return t
}

func defaultWindowBounds(min time.Time, max time.Time) (time.Time, time.Time) {
	return min, max
}
