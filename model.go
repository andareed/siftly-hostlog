package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/andareed/siftly-hostlog/clipboard"
	"github.com/andareed/siftly-hostlog/dialogs"
	"github.com/andareed/siftly-hostlog/logging"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

//var _ tea.Msg = dialogs.SaveRequestedMsg{}

type mode int

const (
	modeView mode = iota
	// modeFilter
	// modeMarking
	modeComment
	modeCommand
	modeTimeWindow
)

//TODO Replace the name renderedRow, as these are not rendered anymore

type model struct {
	viewport            viewport.Model
	drawerPort          viewport.Model
	ready               bool
	cursor              int // index into rows
	lastVisibleRowCount int
	terminalHeight      int
	terminalWidth       int
	pageRowSize         int
	activeDialog        dialogs.Dialog
	fileName            string // filename the data will be saved to
	InitialPath         string
	lastExportFileName  string
	ui                  uiState
	data                dataState
}

func (m *model) InitialiseUI() {
	m.data.showOnlyMarked = false
	m.drawerPort = viewport.New(0, 0)
	m.ui.drawerHeight = 13 // TODO:should be a better way of calcing this rather than hardcoding
	m.ui.drawerOpen = false
	m.ui.mode = modeView
	m.ui.timeWindow = timeWindowUI{
		startInput: initTimeWindowInput(),
		endInput:   initTimeWindowInput(),
		focus:      timeWindowFocusStart,
	}
	m.computeTimeBounds()
	if m.data.timeWindow.Enabled && m.data.hasTimeBounds {
		m.data.timeWindow.Start = clampTimeToBounds(m.data.timeWindow.Start, m.data.timeMin, m.data.timeMax)
		m.data.timeWindow.End = clampTimeToBounds(m.data.timeWindow.End, m.data.timeMin, m.data.timeMax)
		if m.data.timeWindow.Start.After(m.data.timeWindow.End) {
			m.data.timeWindow.Start, m.data.timeWindow.End = defaultWindowBounds(m.data.timeMin, m.data.timeMax)
		}
	}
	if m.data.timeWindow.Enabled {
		m.applyFilter()
	}
}

func (m *model) Init() tea.Cmd {
	m.applyFilter()
	logging.Info("siftly-hostlog: Initialised")
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		logging.Debugf("KEY: %q type=%v mode=%v dialog=%T visible=%v",
			km.String(), km.Type, m.ui.mode,
			m.activeDialog, m.activeDialog != nil && m.activeDialog.IsVisible(),
		)
	}

	logging.Debugf("model:Update called with msg: %#T: %#v", msg, msg)

	if cmd, handled := m.handleSystemMsg(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleDialogInput(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleWindowMsg(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleDialogMsg(msg); handled {
		return m, cmd
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyMsg(keyMsg)
	}
	return m, nil
}

func (m *model) handleSystemMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch fmt.Sprintf("%T", msg) {
	case "cursor.BlinkMsg", "cursor.BlinkCanceledMsg":
		return nil, true
	}
	switch msg := msg.(type) {
	case clearNoticeMsg:
		if msg.id == m.ui.noticeSeq {
			m.ui.noticeMsg = ""
			m.ui.noticeType = ""
		}
		return nil, true
	}
	return nil, false
}

func (m *model) handleDialogInput(msg tea.Msg) (tea.Cmd, bool) {
	if m.activeDialog == nil || !m.activeDialog.IsVisible() {
		return nil, false
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false
	}
	logging.Debugf("DIALOG UPDATE: %T got key %q", m.activeDialog, km.String())
	logging.Debugf("model:Update:: Dialog box is active forward update to it")
	var cmd tea.Cmd
	m.activeDialog, cmd = m.activeDialog.Update(km)
	return cmd, true
}

func (m *model) handleWindowMsg(msg tea.Msg) (tea.Cmd, bool) {
	win, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return nil, false
	}
	m.terminalHeight = win.Height
	m.terminalWidth = win.Width
	m.viewport = viewport.New(0, 0) // TODO: PRetty sure this is redundant
	m.ready = true
	m.refreshView("window-size", true)
	return nil, true
}

func (m *model) handleDialogMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case dialogs.HelpRequestedMsg:
		logging.Infof("Updated was called with msg HelpRequestedMsg (should pop a help dialog box)")
		m.activeDialog = dialogs.NewHelpDialog(Keys.Legend())
		m.activeDialog.Show()
		return nil, true
	case dialogs.SaveRequestedMsg:
		logging.Infof("Update was called with msg SaveRequestedMsg (should pop a dialog box)")
		m.activeDialog = dialogs.NewSaveDialog(defaultSaveName(*m), filepath.Dir(m.fileName))
		m.activeDialog.Show()
		return nil, true
	case dialogs.SaveConfirmedMsg:
		logging.Infof("model:Update::Received SaveConfirmedMsg saving to the current file using path %q", msg.Path)
		m.activeDialog.Hide()
		if err := SaveModel(m, msg.Path); err != nil {
			return m.startNotice("Error", "", noticeDuration), true
		}
		m.fileName = msg.Path
		return m.startNotice("Saved succeeded", "", noticeDuration), true
	case dialogs.SaveCanceledMsg:
		logging.Debugf("model:Update::Received SaveCanceledMsg from dialog and hiding the active dialog")
		m.activeDialog.Hide()
		return nil, true
	case dialogs.ExportRequestedMsg:
		logging.Infof("Update wwas called with msg ExportRequestMsg (pop dialog for exports)")
		m.activeDialog = dialogs.NewExportDialog(defaultExportName(*m), filepath.Dir(m.fileName))
		// TODO: What filename should this default to for an export?
		m.activeDialog.Show()
		return nil, true
	case dialogs.ExportConfirmedMsg:
		logging.Infof("module:Update::ExportConfirmedMsg begin exporting to the file")
		m.activeDialog.Hide()
		if err := ExportModel(m, msg.Path); err != nil {
			return m.startNotice("Export Error", "", noticeDuration), true
		}
		m.lastExportFileName = msg.Path
		return m.startNotice("Exported succeeded", "", noticeDuration), true
	case dialogs.ExportCanceledMsg:
		logging.Debugf("model:Update:: Received ExportCanceledMsg, close down the dialog")
		m.activeDialog.Hide()
		return nil, true
	}
	return nil, false
}

func (m *model) recomputeLayout(height int, width int) {
	// Computes the layout based on whats being rendered
	logging.Debugf("recomputeLayout called with height[%d] width[%d]", height, width)
	height -= 6 // TODO understand this better here? 2 is for the footer, header, plus borders probably
	width -= 6
	if m.ui.drawerOpen {
		drawerContentHeight := 8
		m.ui.drawerHeight = drawerContentHeight + 2
		height -= m.ui.drawerHeight
		m.drawerPort.Width = width // Minus out the padding.
		m.drawerPort.Height = drawerContentHeight
		m.drawerPort.Width = width
	}
	if m.ui.timeWindow.open {
		height -= timeWindowDrawerHeight
	}
	logging.Debugf("Update Received of type Windows Size Message. ViewPort was [%d] and is now getting set to height[%d] width [%d]", m.viewport.Height, height, width)
	m.viewport.Height = height
	m.viewport.Width = width
	m.data.header = layoutColumns(m.data.header, width)
}

func (m *model) refreshView(reason string, withLayout bool) {
	logging.Debugf("refreshView: reason=%s layout=%t", reason, withLayout)
	if withLayout {
		m.recomputeLayout(m.terminalHeight, m.terminalWidth)
	}
	m.clampCursor()
	if m.ui.drawerOpen {
		m.refreshDrawerContent()
	}
	m.viewport.SetContent(m.renderViewport())
}

func (m *model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// if m.activeDialog != nil && m.activeDialog.IsVisible() {
	//
	// return m.handleDialogKey(msg)
	// }
	switch m.ui.mode {
	case modeView:
		return m.handleViewModeKey(msg)
	case modeCommand:
		return m.handleCommandKey(msg)
	case modeTimeWindow:
		return m.handleTimeWindowKey(msg)
	}

	return m, nil
}

func (m *model) handleViewModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	didRefresh := false

	switch {
	// Migrating to a command / input method
	case key.Matches(msg, Keys.TimeWindow):
		m.openTimeWindowDrawer()
		didRefresh = true
	case key.Matches(msg, Keys.JumpToLineNo):
		logging.Infof("Enabling Command: Jumping to specific line number if it exists")
		cmd = m.enterCommand(CmdJump, "", true, false)
	case key.Matches(msg, Keys.Filter):
		logging.Infof("Enabling Command: Filtering")
		cmd = m.enterCommand(CmdFilter, "", true, false)
	case key.Matches(msg, Keys.Search):
		logging.Infof("Enabling Command: Search")
		cmd = m.enterCommand(CmdSearch, "", true, false)
	case key.Matches(msg, Keys.MarkMode):
		logging.Infof("Enable COmmand: Marking")
		cmd = m.enterCommand(CmdMark, "", true, false)
	case key.Matches(msg, Keys.EditComment):
		logging.Infof("Enable COmmand: Edit Comment")
		cmd = m.enterCommand(CmdComment, "", true, false)
	//TODO: Implement Serach
	case key.Matches(msg, Keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, Keys.CopyRow):
		logging.Infof("Key Combination for CopyRow To Clipboard")
		cmd = m.copyRowToClipboard()
	case key.Matches(msg, Keys.JumpToStart):
		logging.Infof("Jumping to start (if filtered will be first row in filter")
		m.jumpToStart()
	case key.Matches(msg, Keys.JumpToEnd):
		logging.Infof("Jumping to end (if filtered will be last row in filter")
		m.jumpToEnd()
	case key.Matches(msg, Keys.ShowMarksOnly):
		// Show Marks only
		logging.Infof("Toggle for Show Marks Only has been pressed")
		m.data.showOnlyMarked = !m.data.showOnlyMarked
		cmd = m.startNotice(fmt.Sprintf("'Show Only Marked Rows' toggled {%b}", m.data.showOnlyMarked), "", noticeDuration)
		m.applyFilter()
	case key.Matches(msg, Keys.NextMark):
		// Next mark jump
		logging.Debug("Here we go; jumping to the next mark")
		m.jumpToNextMark()
	case key.Matches(msg, Keys.PrevMark):
		logging.Debug("Back once again: jumping to the previous mark")
		m.jumpToPreviousMark()
	case key.Matches(msg, Keys.SearchNext):
		if !m.searchNext() {
			cmd = m.startNotice("No matches", "warn", noticeDuration)
		}
	case key.Matches(msg, Keys.SearchPrev):
		if !m.searchPrev() {
			cmd = m.startNotice("No matches", "warn", noticeDuration)
		}
		m.ready = true
	case key.Matches(msg, Keys.ClearFilter):
		// Clear Filter
		logging.Infof("Shift F, clearing Filter")
		m.setFilterPattern("") // Set the filter to nothing which will clear
		cmd = m.startNotice("Cleared filter", "", noticeDuration)
	// case key.Matches(msg, Keys.EditComment):
	// 	// Comment (Edit) if the drawer is open (i.e. C has been pressed previously)
	// 	if m.drawerOpen {
	// 		m.commentInput.Focus()
	// 		m.currentMode = modeComment
	// 	}
	case key.Matches(msg, Keys.ShowComment):
		// Comment in Drawer to be toggled opened / closed
		m.ui.drawerOpen = !m.ui.drawerOpen
		logging.Infof("handleViewModeKey: Toggling Drawer (bottom view above the footer) now see to [%t]", m.ui.drawerOpen)
		m.refreshView("drawer-toggle", true)
		didRefresh = true
	case key.Matches(msg, Keys.RowDown):
		// log.Printf("handleViewModekey: Down or J pressed, moving cursor one position. Cursor [%d] Rows_Total [%d] DrawerOpen[%t]\n", m.cursor, len(m.data.rows), m.ui.drawerOpen)
		if m.cursor < len(m.data.rows)-1 {
			m.cursor++
		}
	case key.Matches(msg, Keys.RowUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, Keys.PageUp):
		m.pageUp()
	case key.Matches(msg, Keys.PageDown):
		m.pageDown()
	case key.Matches(msg, Keys.OpenHelp):
		return m, func() tea.Msg { return dialogs.HelpRequestedMsg{} }
	case key.Matches(msg, Keys.ScrollLeft):
		m.viewport.ScrollLeft(4) // tune step
	case key.Matches(msg, Keys.ScrollRight):
		m.viewport.ScrollRight(4)
	case key.Matches(msg, Keys.SaveToFile):
		return m, func() tea.Msg { return dialogs.SaveRequestedMsg{} }
	case key.Matches(msg, Keys.ExportToFile):
		return m, func() tea.Msg { return dialogs.ExportRequestedMsg{} }
	}

	//TODO: DON'T THINK WE SHOULD BE RENDERING TABLE EVERY TIME TBH
	if m.ready && !didRefresh {
		m.refreshView("view-key", false)
	}
	return m, cmd
}

func (m *model) copyRowToClipboard() tea.Cmd {
	if m.cursor >= 0 && m.cursor < len(m.data.filteredIndices) {
		row := m.data.rows[m.data.filteredIndices[m.cursor]]
		text := row.Join("\t") // Tab Delimetered string
		if err := clipboard.Copy(text); err != nil {
			return m.startNotice("Error with Clipboard occurred.", "", noticeDuration)
		} else {
			return m.startNotice("Copied Row COntent to Clipboard", "", noticeDuration)
		}
	}
	return nil
}

func (m *model) pageDown() {
	if len(m.data.filteredIndices) == 0 {
		return
	}
	if m.cursor+m.pageRowSize < len(m.data.filteredIndices) {
		m.cursor += m.pageRowSize
	} else {
		m.cursor = len(m.data.filteredIndices) - 1
	}
}

func (m *model) pageUp() {
	if len(m.data.filteredIndices) == 0 {
		return
	}
	m.cursor -= m.pageRowSize
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *model) clampCursor() {
	if len(m.data.filteredIndices) == 0 {
		m.cursor = -1
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.data.filteredIndices) {
		m.cursor = len(m.data.filteredIndices) - 1
	}
}

func (m *model) currentRowHashID() uint64 {
	if len(m.data.filteredIndices) == 0 || m.cursor < 0 || m.cursor >= len(m.data.filteredIndices) {
		logging.Debugf("currentRowHashID called but no filteredIndices available (cursor=%d, len=%d)", m.cursor, len(m.data.filteredIndices))
		return 0 // or some sentinel value
	}
	rowIdx := m.data.filteredIndices[m.cursor]
	hashId := m.data.rows[rowIdx].id
	logging.Debugf("currentRowHashID called returning HashID[%d] for cursor[%d] at filteredIndex[%d] which maps to rowIndex[%d]", hashId, m.cursor, rowIdx, rowIdx)
	return hashId
}

func (m *model) jumpToHashID(hashId uint64) {
	if hashId == 0 {
		logging.Debugf("jumpToHashID called with HashID of 0, so returning")
		m.cursor = 0
		return
	}

	logging.Debugf("jumpToHashID called looking for HashID[%d]", hashId)
	for i, idx := range m.data.filteredIndices {
		if m.data.rows[idx].id == hashId {
			m.cursor = i
			logging.Debugf("jumpToHashID: Jumping to index [%d] for hashID[%d]", i, hashId)
			return
		}
	}

	m.cursor = 0
	logging.Warnf("jumpToHashID: No match found for hashID[%d] so setting cursor to 0", hashId)
}

func (m *model) checkViewPortHasData() bool {
	if len(m.data.filteredIndices) == 0 {
		logging.Debug("filterIndicies is empty")
		return false
	}
	if m.cursor < 0 {
		logging.Debug("Cursor at 0 or below")
		return false
	}
	return true
}
func (m *model) applyFilter() {
	logging.Debugf("applyFilter called")
	// Remember the Hash of what we have current selected

	currentRowHash := m.currentRowHashID()              // should be called before we reset the filteredIndices
	m.data.filteredIndices = m.data.filteredIndices[:0] // reset slice

	if m.data.filterRegex == nil && !m.data.showOnlyMarked && !m.data.timeWindow.Enabled {
		logging.Debug("applyFilter: No filter text and showOnly marked is false there all indices being added to filteredIncidices")
		// Maybe used clamp?
		for i := range m.data.rows {
			m.data.filteredIndices = append(m.data.filteredIndices, i)
		}
		if len(m.data.filteredIndices) == 0 {
			m.cursor = 0
		}
		m.jumpToHashID(currentRowHash)
		return
	}

	for i, row := range m.data.rows {
		if m.includeRow(row, i) {
			m.data.filteredIndices = append(m.data.filteredIndices, i)
		}
	}

	if len(m.data.filteredIndices) == 0 {
		// No matches found prevent index panics
		m.cursor = -1
	}

	m.jumpToHashID(currentRowHash)
	m.clampCursor()
}

// endregion

func defaultExportName(m model) string {
	if m.lastExportFileName != "" {
		return m.lastExportFileName
	}

	base := strings.TrimSuffix(m.InitialPath, filepath.Ext(m.InitialPath))
	return "export-" + base + ".csv"

}

func defaultSaveName(m model) string {
	// Case 1: we already have a current filename — use it
	if m.fileName != "" {
		return m.fileName
	}

	// Case 2: otherwise fall back to whatever initial path we have
	initial := m.InitialPath
	if initial == "" {
		return "output.json" // final fallback
	}

	// Case 3: if the initial path already ends with .json, use it
	if strings.HasSuffix(strings.ToLower(initial), ".json") {
		return initial
	}

	// Case 4: replace any existing extension with .json
	base := strings.TrimSuffix(initial, filepath.Ext(initial))
	return base + ".json"
}
