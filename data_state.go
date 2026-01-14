package main

import (
	"regexp"
	"time"
)

type TimeWindow struct {
	Enabled bool
	Start   time.Time
	End     time.Time
}

type dataState struct {
	header          []ColumnMeta // single row for column titles in headerview
	rows            []renderedRow
	markedRows      map[uint64]MarkColor // map row index to color code
	commentRows     map[uint64]string    // map row index to string to store comments
	showOnlyMarked  bool
	filterRegex     *regexp.Regexp
	filteredIndices []int // to store the list of indicides that match the current regex
	timeWindow      TimeWindow
	timeMin         time.Time
	timeMax         time.Time
	hasTimeBounds   bool
	timeColumnIndex int
	rowTimes        []time.Time
	rowHasTimes     []bool
}
