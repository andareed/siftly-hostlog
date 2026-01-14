package main

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

const (
	timeWindowFocusStart = iota
	timeWindowFocusEnd
	timeWindowFocusScrubber
)

const (
	timeWindowDrawerContentHeight = 5
	timeWindowDrawerHeight        = timeWindowDrawerContentHeight + 2
	timeWindowStepMin             = 15 * time.Minute
	timeWindowStepDefault         = 30 * time.Minute
	timeWindowStepMax             = 2 * time.Hour
)

type timeWindowUI struct {
	open       bool
	focus      int
	startInput textinput.Model
	endInput   textinput.Model
	errorMsg   string
	draftStart time.Time
	draftEnd   time.Time
	origWindow TimeWindow
	step       time.Duration
}

func initTimeWindowInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = timeInputLayout
	ti.CharLimit = len(timeInputLayout)
	ti.Width = len(timeInputLayout)
	ti.Prompt = ""
	return ti
}
