package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andareed/siftly-hostlog/logging"
	tea "github.com/charmbracelet/bubbletea"
)

var logFile = flag.String("debug", "", "Write Debug Logs to file")

func main() {
	// Define flags
	flag.Parse()
	cleanup, err := logging.SetupLogging(*logFile)
	if err != nil {
		log.Fatalf("Failed to setup logging %v", err)
	}
	defer cleanup()

	log.Println("siftly-hostlog: Started")
	// After flags, the remaining args are positional
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: sfhost [--debug debug.log] <file.log>")
		os.Exit(1)
	}
	csvFile := args[0]

	// Open CSV
	f, err := os.Open(csvFile)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}
	log.Printf("Read %d records from %s", len(records), csvFile)

	// Start TUI
	_, err = tea.NewProgram(initialModel(records), tea.WithAltScreen()).Run()
	if err != nil {
		log.Printf("Tea program error: %v", err)
		fmt.Println("Error:", err)
	}
}
