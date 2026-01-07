package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
		log.Fatalf("Failed to setup logging %v", err)
	}
	defer cleanup()

	log.Println("siftly-hostlog: Started")

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: sfhost [--debug debug.log] <file.csv|file.json>")
		os.Exit(1)
	}

	inputPath := args[0]

	m, err := loadModelAuto(inputPath)
	if err != nil {
		log.Fatalf("failed to load %q: %v", inputPath, err)
	}

	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		log.Printf("Tea program error: %v", err)
		fmt.Println("Error:", err)
	}
}

func loadModelAuto(path string) (*model, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return newModelFromJSONFile(path)
	case ".csv":
		return newModelFromCSVFile(path)
	default:
		return nil, fmt.Errorf("unsupported file extension %q (want .csv or .json)", ext)
	}
}

// Load Data From Serialized JSONs using LoadModel(m, path)
// Implies that this has been analysed previously and saved
func newModelFromJSONFile(path string) (*model, error) {
	m := &model{}
	if err := LoadModel(m, path); err != nil {
		return nil, err
	}
	m.InitialPath = path
	m.InitialiseUI()
	return m, nil
}

func newModelFromCSVFile(path string) (*model, error) {
	// ...read csv...
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV %q has no rows", path)
	}

	m := initialModelFromCSV(records)
	m.InitialPath = path
	m.InitialiseUI()
	return m, nil
}

// Builds rows from CSV
func initialModelFromCSV(data [][]string) *model {

	rawHeader := data[0]
	cols := make([]ColumnMeta, len(rawHeader))
	for i, name := range rawHeader {
		role := detectRole(name)
		cols[i] = ColumnMeta{
			Name:     name,
			Index:    i,
			Role:     role,
			Visible:  true,
			MinWidth: defaultMinWidthForRole(role),
			Weight:   defaultWeightForRole(role),
		}
	}

	rows := make([]renderedRow, 0, len(data)-1)

	hasData := make([]bool, len(cols)) // track which columns ever have content

	for i, csvRow := range data[1:] {
		// Track whether this row has data in each column
		for colIdx := range cols {
			if colIdx < len(csvRow) && strings.TrimSpace(csvRow[colIdx]) != "" {
				hasData[colIdx] = true
			}
		}

		// Build your row like normal
		row := renderedRow{
			cols:   csvRow,
			height: 1,
		}
		row.id = row.ComputeID()
		row.originalIndex = i + 1

		rows = append(rows, row)
	}

	// After building rows: mark empty columns
	for i := range cols {
		if !hasData[i] {
			// Entire column was empty → hide it (unless it's your primary/details column)
			if cols[i].Role != RolePrimary {
				log.Printf("No column data in none-Primary colum: %d so setting visibility to false \n", i)
				cols[i].Visible = false
			}
			cols[i].Weight = 0
			cols[i].Width = 0
		}
	}

	return &model{
		header:      cols,
		rows:        rows,
		currentMode: modView,

		markedRows:  make(map[uint64]MarkColor),
		commentRows: make(map[uint64]string),

		// the rest is initialized in initUIBits
	}
}
