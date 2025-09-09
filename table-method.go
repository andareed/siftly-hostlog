package main

// go get github.com/charmbracelet/bubbletea
// go get github.com/charmbracelet/lipgloss
// go get github.com/charmbracelet/charm
// go get github.com/muesli/reflow

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

// ENUMS
type (
	state int
	color int
)

var (
	redStyle   = lipgloss.NewStyle().Background(lipgloss.Color("9"))
	greenStyle = lipgloss.NewStyle().Background(lipgloss.Color("10"))
	amberStyle = lipgloss.NewStyle().Background(lipgloss.Color("11"))
)

const (
	columnTime = iota
	columnHost
	columnDetails
	columnAppliance
	columnMAC
	columnIPv6
	columnComment
)

const (
	colorNone color = iota
	colorRed
	colorGreen
	colorAmber
)

const (
	stateNormal state = iota
	stateMarking
	stateCommenting
	stateFiltering
)

type Row struct {
	original   []string
	color      color
	comment    string
	filteredIn bool
	id         int
}

type ViewModel struct {
	table         table.Model
	textarea      textarea.Model
	textinput     textinput.Model
	rows          []Row
	state         state
	showMarked    bool
	textFilter    string
	quitting      bool
	height        int
	weight        int
	activeFilters []string
	statusMessage string
}

// View implements tea.Model.
func (m ViewModel) View() string {
	if m.quitting {
		return ""
	}

	header := ""
	//TODO; ACtive Filters

	footer := ""
	legend := "q: quit"
	footer += legend

	if m.statusMessage != "" {
		footer += "\n" + m.statusMessage
	}
	return fmt.Sprintf("%s\n%s\n%s", header, m.table.View(), footer)
}

func (m ViewModel) Init() tea.Cmd {
	return nil
}

func (m ViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	//TODO: Flesh this method out
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
		}
	}
	// Handles the default messages being passed to the table
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// main
func tableMethod() {
	// TODO: Maybe show have a parse arguments as the first function?
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[1]
	} else {
		fmt.Printf("Filename argument has not been passed \n")
		os.Exit(1)
	}

	fmt.Printf("Hello, world is my file %s\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	r := csv.NewReader(f)
	fmt.Println("Header Read")
	// Headers
	header, err := r.Read()
	if err != nil {
		fmt.Printf("Error reading head from csv file: %v\n", err)
	}

	// TODO: Check that comment does not exist before we addit
	header = append(header, "Comment")

	fmt.Println("Columns")
	// Source Log Headers
	columns := []table.Column{
		{Title: "Time", Width: 25},
		{Title: "Host", Width: 25},
		{Title: "Details", Width: 80},
		{Title: "Appliance", Width: 25},
		{Title: "MAC Address", Width: 25},
		{Title: "Ipv6 Address", Width: 25},
		{Title: "Comment", Width: 80},
	}

	//TODO: Switch to using column
	_ = columns

	var records [][]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		records = append(records, record)
	}

	rows := make([]Row, len(records))
	for i, record := range records {
		rows[i] = Row{original: record, filteredIn: true, id: i}
	}

	// Style
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(toTableRows(rows, 80, 80)),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	t.SetStyles(s)

	// Setting up text inputs

	filterTextInput := textinput.New()
	filterTextInput.Placeholder = "Enter regex filter..."

	commentTextArea := textarea.New()
	commentTextArea.Placeholder = "Enter Comment for this row..."

	m := ViewModel{
		table:     t,
		textarea:  commentTextArea,
		textinput: filterTextInput,
		rows:      rows,
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	fmt.Println("End of the road..")
}

func toTableRows2(rows []Row, detailsWidth, commentWidth int) []table.Row {
	var tableRows []table.Row
	for _, row := range rows {
		if row.filteredIn {
			// TODO: Completed filtered
			content := wordwrap.String(row.original[columnDetails], detailsWidth)
			comment := wordwrap.String(row.comment, commentWidth)
			tableRow := table.Row{
				row.original[columnTime],
				row.original[columnHost],
				content,
				row.original[columnAppliance],
				row.original[columnMAC],
				row.original[columnIPv6],
				comment,
			}
			tableRows = append(tableRows, tableRow)
		}
	}
	return tableRows
}

func toTableRows(rows []Row, detailsWidth int, commentWidth int) []table.Row {
	var tableRows []table.Row

	for _, r := range rows {
		original := r.original

		if len(original) < columnIPv6+1 {
			continue
		}

		// Wrap long fields
		detailsWrapped := strings.Split(wordwrap.String(original[columnDetails], detailsWidth), "\n")
		commentWrapped := strings.Split(wordwrap.String(r.comment, commentWidth), "\n")

		maxLines := max(len(detailsWrapped), len(commentWrapped))
		// Pad each field to match maxLines
		timeLines := padLines([]string{original[columnTime]}, maxLines)
		hostLines := padLines([]string{original[columnHost]}, maxLines)
		detailsLines := padLines(detailsWrapped, maxLines)
		applianceLines := padLines([]string{original[columnAppliance]}, maxLines)
		macLines := padLines([]string{original[columnMAC]}, maxLines)
		ipv6Lines := padLines([]string{original[columnIPv6]}, maxLines)
		commentLines := padLines(commentWrapped, maxLines)
		_ = commentLines
		commentLines2 := "hello\nworldi1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111110"
		// Join lines with \n to allow vertical alignment in Bubbletea
		tableRow := table.Row{
			strings.Join(timeLines, "\n"),
			strings.Join(hostLines, "\n"),
			strings.Join(detailsLines, "\n"),
			strings.Join(applianceLines, "\n"),
			strings.Join(macLines, "\n"),
			strings.Join(ipv6Lines, "\n"),
			commentLines2,
		}

		tableRows = append(tableRows, tableRow)
	}

	return tableRows
}

func padLines(lines []string, target int) []string {
	for len(lines) < target {
		lines = append(lines, "")
	}
	return lines
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
