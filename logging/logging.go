package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SetupLogging configures logging.
// If filename is empty, logging is disabled (except log.Fatal/panic).
// If filename is set, logs go to that file and Bubble Tea logs are enabled too.
func SetupLogging(filename string) (cleanup func(), err error) {
	SetLevelFromEnv()
	if filename == "" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.SetOutput(io.Discard) // <- key change
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

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

var currentLevel = LevelInfo

func SetLevelFromEnv() {
	if v := strings.TrimSpace(os.Getenv("SIFTLY_LOG_LEVEL")); v != "" {
		currentLevel = levelFromString(v)
	}
}

func levelFromString(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "off", "none":
		return LevelOff
	default:
		return LevelInfo
	}
}

func Debugf(format string, args ...any) { logf(LevelDebug, "DEBUG", format, args...) }
func Infof(format string, args ...any)  { logf(LevelInfo, "INFO", format, args...) }
func Warnf(format string, args ...any)  { logf(LevelWarn, "WARN", format, args...) }
func Errorf(format string, args ...any) { logf(LevelError, "ERROR", format, args...) }

func Debug(args ...any) { logf(LevelDebug, "DEBUG", "%s", fmt.Sprint(args...)) }
func Info(args ...any)  { logf(LevelInfo, "INFO", "%s", fmt.Sprint(args...)) }
func Warn(args ...any)  { logf(LevelWarn, "WARN", "%s", fmt.Sprint(args...)) }
func Error(args ...any) { logf(LevelError, "ERROR", "%s", fmt.Sprint(args...)) }

func Fatalf(format string, args ...any) {
	log.Fatalf("[FATAL] "+format, args...)
}

func logf(level Level, prefix, format string, args ...any) {
	if currentLevel == LevelOff || level < currentLevel {
		return
	}
	log.Printf("["+prefix+"] "+format, args...)
}
