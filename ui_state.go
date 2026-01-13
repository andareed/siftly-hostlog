package main

type uiState struct {
	mode         mode
	command      CommandInput
	drawerOpen   bool
	drawerHeight int
	noticeMsg    string
	noticeType   string
	noticeSeq    int
	searchQuery  string
}
