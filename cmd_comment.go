package main

import (
	"github.com/andareed/siftly-hostlog/logging"
)

func (m *model) addComment(comment string) {
	logging.Debug("CommentCurrent called..")
	if (m.cursor) < 0 || m.cursor >= len(m.data.filteredIndices) {
		return
	}

	idx := m.data.filteredIndices[m.cursor]
	hashId := m.data.rows[idx].id
	if comment == "" {
		delete(m.data.commentRows, hashId)
		logging.Infof("Clear comment Index[%d] on HashID[%d]", idx, hashId)
		return
		//TODO: Probably need this sending a notificatoin
	}
	m.data.commentRows[hashId] = comment
	logging.Infof("Setting Comment[%s] to Index[%d] on HashID[%d]", comment, idx, hashId)
}

func (m *model) getCommentContent(rowIdx uint64) string {
	// Probably want some error checking around the rowIdx
	if c, ok := m.data.commentRows[rowIdx]; ok && c != "" {
		return c
	}
	return "" // No comment, so returning blank
}

func (m *model) refreshDrawerContent() {
	logging.Debug("refreshDrawerContent called..")
	currentComment := m.getCommentContent(m.currentRowHashID())
	logging.Debugf("Comment Input and Drawer Port being set to: %s", currentComment)
	m.drawerPort.SetContent(currentComment)
}
