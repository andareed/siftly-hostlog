package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type clearNoticeMsg struct{ id int }

const noticeDuration = 2 * time.Second

func noticeText(msg, kind string) string {
	if msg == "" {
		return ""
	}
	var icon string
	switch kind {
	case "info":
		icon = "ℹ"
	case "success":
		icon = "✓"
	case "warn":
		icon = "!"
	case "error":
		icon = "×"
	default:
		icon = ""
	}
	if icon == "" {
		return msg
	}
	return icon + " " + msg
}

func (m *model) startNotice(msg, msgType string, d time.Duration) tea.Cmd {
	// set current notice
	m.noticeMsg = msg
	m.noticeType = msgType

	// bump sequence to invalidate older timers
	m.noticeSeq++
	id := m.noticeSeq

	// schedule a clear for this specific notice id
	return tea.Tick(d, func(time.Time) tea.Msg { return clearNoticeMsg{id: id} })
}
