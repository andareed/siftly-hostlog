package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/andareed/siftly-hostlog/logging"
	tea "github.com/charmbracelet/bubbletea"
)

var logFile = flag.String("debug", "", "Write Debug Logs to file")

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")

	flag.Parse()

	// --- EARLY EXIT ---
	if *versionFlag {
		fmt.Println("Version:", Version)
		os.Exit(0)
	}

	// Anything below here should NOT run if --version was provided.
	cleanup, err := logging.SetupLogging(*logFile)
	if err != nil {
		logging.Fatalf("Failed to setup logging %v", err)
	}
	defer cleanup()

	logging.Info("siftly-hostlog: Started")

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: sfhost [--debug debug.log] <file.csv|file.json>")
		os.Exit(1)
	}

	inputPath := args[0]

	m, err := loadModelAuto(inputPath)
	if err != nil {
		logging.Fatalf("failed to load %q: %v", inputPath, err)
	}

	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		logging.Errorf("Tea program error: %v", err)
		fmt.Println("Error:", err)
	}
}
