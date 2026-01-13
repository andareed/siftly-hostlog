package main

import (
	"log"
)

func (m *model) addComment(comment string) {
	log.Printf("CommentCurrent called..\n")
	if (m.cursor) < 0 || m.cursor >= len(m.filteredIndices) {
		return
	}

	idx := m.filteredIndices[m.cursor]
	hashId := m.rows[idx].id
	if comment == "" {
		delete(m.commentRows, hashId)
		log.Printf("Clear comment Index[%d] on HashID[%d]\n", idx, hashId)
		return
		//TODO: Probably need this sending a notificatoin
	}
	m.commentRows[hashId] = comment
	log.Printf("Setting Comment[%s] to Index[%d] on HashID[%d]\n", comment, idx, hashId)
}

func (m *model) getCommentContent(rowIdx uint64) string {
	// Probably want some error checking around the rowIdx
	if c, ok := m.commentRows[rowIdx]; ok && c != "" {
		return c
	}
	return "" // No comment, so returning blank
}

func (m *model) refreshDrawerContent() {
	log.Printf("refreshDrawerContent called..")
	currentComment := m.getCommentContent(m.currentRowHashID())
	log.Printf("Comment Input and Drawer Port being set to: %s", currentComment)
	m.commentInput.SetValue(currentComment)
	m.drawerPort.SetContent(currentComment)
}
