package main

import "regexp"

type dataState struct {
	header          []ColumnMeta // single row for column titles in headerview
	rows            []renderedRow
	markedRows      map[uint64]MarkColor // map row index to color code
	commentRows     map[uint64]string    // map row index to string to store comments
	showOnlyMarked  bool
	filterRegex     *regexp.Regexp
	filteredIndices []int // to store the list of indicides that match the current regex
}
