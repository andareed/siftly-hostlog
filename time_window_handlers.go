package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) openTimeWindowDrawer() {
	tw := &m.ui.timeWindow
	tw.open = true
	tw.errorMsg = ""
	tw.origWindow = m.data.timeWindow
	tw.step = timeWindowStepDefault

	if !m.data.hasTimeBounds {
		tw.errorMsg = "No timestamps available"
		tw.startInput.SetValue("")
		tw.endInput.SetValue("")
		tw.draftStart = time.Time{}
		tw.draftEnd = time.Time{}
		m.setTimeWindowFocus(timeWindowFocusStart)
		m.ui.mode = modeTimeWindow
		m.refreshView("time-window-open", true)
		return
	}

	if m.data.timeWindow.Start.IsZero() || m.data.timeWindow.End.IsZero() {
		tw.draftStart, tw.draftEnd = defaultWindowBounds(m.data.timeMin, m.data.timeMax)
	} else {
		tw.draftStart = m.data.timeWindow.Start
		tw.draftEnd = m.data.timeWindow.End
	}

	m.updateTimeWindowInputsFromDraft()
	m.setTimeWindowFocus(timeWindowFocusStart)
	m.ui.mode = modeTimeWindow
	m.refreshView("time-window-open", true)
}

func (m *model) closeTimeWindowDrawer() {
	m.ui.timeWindow.open = false
	m.ui.timeWindow.errorMsg = ""
	m.ui.mode = modeView
	m.refreshView("time-window-close", true)
}

func (m *model) handleTimeWindowKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tw := &m.ui.timeWindow

	switch {
	case msg.Type == tea.KeyEsc:
		m.closeTimeWindowDrawer()
		return m, nil
	case msg.Type == tea.KeyEnter:
		m.applyTimeWindowFromInputs()
		return m, nil
	case msg.String() == "r":
		m.resetTimeWindowDraft()
		return m, nil
	case msg.Type == tea.KeyTab:
		m.setTimeWindowFocus((tw.focus + 1) % 3)
		return m, nil
	case msg.Type == tea.KeyShiftTab:
		m.setTimeWindowFocus((tw.focus + 2) % 3)
		return m, nil
	case tw.focus == timeWindowFocusScrubber && msg.Type == tea.KeyLeft:
		m.shiftTimeWindow(-m.timeWindowStep())
		return m, nil
	case tw.focus == timeWindowFocusScrubber && msg.Type == tea.KeyRight:
		m.shiftTimeWindow(m.timeWindowStep())
		return m, nil
	case tw.focus == timeWindowFocusScrubber && msg.Type == tea.KeyShiftLeft:
		m.expandTimeWindow(-m.timeWindowStep())
		return m, nil
	case tw.focus == timeWindowFocusScrubber && msg.Type == tea.KeyShiftRight:
		m.expandTimeWindow(m.timeWindowStep())
		return m, nil
	case tw.focus == timeWindowFocusScrubber && msg.String() == "-":
		m.adjustTimeWindowStep(false)
		return m, nil
	case tw.focus == timeWindowFocusScrubber && (msg.String() == "+" || msg.String() == "="):
		m.adjustTimeWindowStep(true)
		return m, nil
	}

	var cmd tea.Cmd
	if tw.focus == timeWindowFocusStart {
		tw.startInput, cmd = tw.startInput.Update(msg)
		return m, cmd
	}
	if tw.focus == timeWindowFocusEnd {
		tw.endInput, cmd = tw.endInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) setTimeWindowFocus(focus int) {
	tw := &m.ui.timeWindow
	tw.focus = focus
	switch focus {
	case timeWindowFocusStart:
		tw.startInput.Focus()
		tw.endInput.Blur()
	case timeWindowFocusEnd:
		tw.startInput.Blur()
		tw.endInput.Focus()
	default:
		tw.startInput.Blur()
		tw.endInput.Blur()
	}
}

func (m *model) updateTimeWindowInputsFromDraft() {
	tw := &m.ui.timeWindow
	if !tw.draftStart.IsZero() {
		tw.startInput.SetValue(tw.draftStart.Format(timeInputLayout))
	}
	if !tw.draftEnd.IsZero() {
		tw.endInput.SetValue(tw.draftEnd.Format(timeInputLayout))
	}
}

func (m *model) syncDraftFromInputs() {
	tw := &m.ui.timeWindow
	if !m.data.hasTimeBounds {
		return
	}
	loc := m.data.timeMax.Location()
	startStr := strings.TrimSpace(tw.startInput.Value())
	endStr := strings.TrimSpace(tw.endInput.Value())
	start, err := time.ParseInLocation(timeInputLayout, startStr, loc)
	if err == nil {
		tw.draftStart = start
	}
	end, err := time.ParseInLocation(timeInputLayout, endStr, loc)
	if err == nil {
		tw.draftEnd = end
	}
}

func (m *model) resetTimeWindowDraft() {
	tw := &m.ui.timeWindow
	tw.errorMsg = ""

	if !m.data.hasTimeBounds {
		tw.errorMsg = "No timestamps available"
		return
	}

	tw.draftStart, tw.draftEnd = defaultWindowBounds(m.data.timeMin, m.data.timeMax)
	m.updateTimeWindowInputsFromDraft()

	if timeWindowResetMode == timeWindowResetDisable {
		m.data.timeWindow.Enabled = false
		m.applyFilter()
	}
}

func (m *model) applyTimeWindowFromInputs() {
	tw := &m.ui.timeWindow
	tw.errorMsg = ""

	if !m.data.hasTimeBounds {
		tw.errorMsg = "No timestamps available"
		return
	}

	loc := m.data.timeMax.Location()
	startStr := strings.TrimSpace(tw.startInput.Value())
	endStr := strings.TrimSpace(tw.endInput.Value())

	start, err := time.ParseInLocation(timeInputLayout, startStr, loc)
	if err != nil {
		tw.errorMsg = "Invalid start time"
		return
	}
	end, err := time.ParseInLocation(timeInputLayout, endStr, loc)
	if err != nil {
		tw.errorMsg = "Invalid end time"
		return
	}
	if start.After(end) {
		tw.errorMsg = "Start is after end"
		return
	}

	start = clampTimeToBounds(start, m.data.timeMin, m.data.timeMax)
	end = clampTimeToBounds(end, m.data.timeMin, m.data.timeMax)
	if start.After(end) {
		tw.errorMsg = "Start is after end"
		return
	}

	m.data.timeWindow = TimeWindow{
		Enabled: true,
		Start:   start,
		End:     end,
	}
	tw.draftStart = start
	tw.draftEnd = end
	m.applyFilter()
	m.closeTimeWindowDrawer()
}

func (m *model) shiftTimeWindow(delta time.Duration) {
	tw := &m.ui.timeWindow
	tw.errorMsg = ""

	if !m.data.hasTimeBounds {
		tw.errorMsg = "No timestamps available"
		return
	}

	m.syncDraftFromInputs()
	if tw.draftStart.IsZero() || tw.draftEnd.IsZero() {
		tw.draftStart, tw.draftEnd = defaultWindowBounds(m.data.timeMin, m.data.timeMax)
	}

	min := m.data.timeMin
	max := m.data.timeMax
	rangeDur := max.Sub(min)
	windowDur := tw.draftEnd.Sub(tw.draftStart)
	if windowDur <= 0 {
		windowDur = timeWindowStepMin
	}
	if windowDur > rangeDur {
		tw.draftStart = min
		tw.draftEnd = max
		m.updateTimeWindowInputsFromDraft()
		return
	}

	nextStart := tw.draftStart.Add(delta)
	nextEnd := tw.draftEnd.Add(delta)
	if nextStart.Before(min) {
		nextStart = min
		nextEnd = min.Add(windowDur)
	}
	if nextEnd.After(max) {
		nextEnd = max
		nextStart = max.Add(-windowDur)
	}

	tw.draftStart = nextStart
	tw.draftEnd = nextEnd
	m.updateTimeWindowInputsFromDraft()
}

func (m *model) timeWindowStep() time.Duration {
	step := m.ui.timeWindow.step
	if step <= 0 {
		return timeWindowStepDefault
	}
	if step < timeWindowStepMin {
		return timeWindowStepMin
	}
	if step > timeWindowStepMax {
		return timeWindowStepMax
	}
	return step
}

func (m *model) adjustTimeWindowStep(increase bool) {
	step := m.timeWindowStep()
	if increase {
		step *= 2
	} else {
		step /= 2
	}
	if step < timeWindowStepMin {
		step = timeWindowStepMin
	}
	if step > timeWindowStepMax {
		step = timeWindowStepMax
	}
	m.ui.timeWindow.step = step
}

func (m *model) expandTimeWindow(delta time.Duration) {
	tw := &m.ui.timeWindow
	tw.errorMsg = ""

	if !m.data.hasTimeBounds {
		tw.errorMsg = "No timestamps available"
		return
	}

	m.syncDraftFromInputs()
	if tw.draftStart.IsZero() || tw.draftEnd.IsZero() {
		tw.draftStart, tw.draftEnd = defaultWindowBounds(m.data.timeMin, m.data.timeMax)
	}

	min := m.data.timeMin
	max := m.data.timeMax
	if delta < 0 {
		nextStart := tw.draftStart.Add(delta)
		if nextStart.Before(min) {
			nextStart = min
		}
		tw.draftStart = nextStart
		if tw.draftStart.After(tw.draftEnd) {
			tw.draftEnd = tw.draftStart
		}
	} else if delta > 0 {
		nextEnd := tw.draftEnd.Add(delta)
		if nextEnd.After(max) {
			nextEnd = max
		}
		tw.draftEnd = nextEnd
		if tw.draftEnd.Before(tw.draftStart) {
			tw.draftStart = tw.draftEnd
		}
	}

	m.updateTimeWindowInputsFromDraft()
}

func formatStep(step time.Duration) string {
	if step%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(step/time.Hour))
	}
	return fmt.Sprintf("%dm", int(step/time.Minute))
}

func (m *model) timeWindowDrawerView(width int) string {
	tw := &m.ui.timeWindow
	innerWidth := max(0, width-2)
	lineStyle := lipgloss.NewStyle().Width(innerWidth)

	startLine := fmt.Sprintf("Start: %s", tw.startInput.View())
	endLine := fmt.Sprintf("End:   %s", tw.endInput.View())
	scrubberLine := m.timeWindowScrubberLine(innerWidth)
	helpLine := fmt.Sprintf("t: open  tab: next  enter: apply  r: reset  esc: cancel  ←/→: move %s  shift+←/→: expand %s  -/+: step",
		formatStep(m.timeWindowStep()),
		formatStep(m.timeWindowStep()),
	)
	errorLine := ""
	if tw.errorMsg != "" {
		errorLine = "Error: " + tw.errorMsg
	}

	lines := []string{
		lineStyle.Render(startLine),
		lineStyle.Render(endLine),
		lineStyle.Render(scrubberLine),
		lineStyle.Render(helpLine),
		lineStyle.Render(errorLine),
	}

	content := strings.Join(lines, "\n")
	return timeWindowArea.Width(width).Render(content)
}

func (m *model) timeWindowScrubberLine(width int) string {
	if !m.data.hasTimeBounds {
		return "Scrubber: n/a"
	}

	start := m.ui.timeWindow.draftStart
	end := m.ui.timeWindow.draftEnd
	if start.IsZero() || end.IsZero() {
		start, end = defaultWindowBounds(m.data.timeMin, m.data.timeMax)
	}

	minLabel := m.data.timeMin.Format(timeInputLayout)
	maxLabel := m.data.timeMax.Format(timeInputLayout)
	padding := 2
	barWidth := width - len(minLabel) - len(maxLabel) - padding*2
	if barWidth < 10 {
		return fmt.Sprintf("Window: %s - %s", start.Format(timeInputLayout), end.Format(timeInputLayout))
	}

	bar := make([]rune, barWidth)
	for i := range bar {
		bar[i] = '-'
	}
	rangeDur := m.data.timeMax.Sub(m.data.timeMin)
	if rangeDur <= 0 {
		return "Scrubber: n/a"
	}

	windowStart := clampTimeToBounds(start, m.data.timeMin, m.data.timeMax)
	windowEnd := clampTimeToBounds(end, m.data.timeMin, m.data.timeMax)
	startPos := int(float64(barWidth-1) * windowStart.Sub(m.data.timeMin).Seconds() / rangeDur.Seconds())
	endPos := int(float64(barWidth-1) * windowEnd.Sub(m.data.timeMin).Seconds() / rangeDur.Seconds())
	if startPos < 0 {
		startPos = 0
	}
	if endPos >= barWidth {
		endPos = barWidth - 1
	}
	if endPos < startPos {
		startPos, endPos = endPos, startPos
	}
	for i := startPos; i <= endPos; i++ {
		bar[i] = '='
	}
	if startPos >= 0 && startPos < barWidth {
		bar[startPos] = '['
	}
	if endPos >= 0 && endPos < barWidth {
		bar[endPos] = ']'
	}

	return fmt.Sprintf("%s  %s  %s", minLabel, string(bar), maxLabel)
}

func (m *model) timeWindowStatusLabel() string {
	if !m.data.timeWindow.Enabled {
		return "Window: off"
	}
	return fmt.Sprintf("Window: %s - %s",
		m.data.timeWindow.Start.Format(timeInputLayout),
		m.data.timeWindow.End.Format(timeInputLayout),
	)
}
