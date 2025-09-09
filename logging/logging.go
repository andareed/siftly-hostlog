package logging

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// SetupLogging configures logging.
// If filename is empty, logging is disabled (except log.Fatal/panic).
// If filename is set, logs go to that file and Bubble Tea logs are enabled too.
func SetupLogging(filename string) (cleanup func(), err error) {
	if filename == "" {
		// no logging
		return func() {}, nil
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// configure stdlib logger
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// configure Bubble Tea logger
	tf, err := tea.LogToFile(filename, "debug")
	if err != nil {
		f.Close()
		return nil, err
	}

	// cleanup closes both files
	cleanup = func() {
		tf.Close()
		f.Close()
	}
	return cleanup, nil
}
