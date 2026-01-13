package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// --- Wire format ---

const snapshotVersion = 1

type renderedRowDTO struct {
	Cols          []string `json:"cols"`
	Height        int      `json:"height"`
	ID            uint64   `json:"id"`
	OriginalIndex int      `json:"originalIndex"`
}

type snapshotDTO struct {
	Version  int               `json:"version"`
	Header   []ColumnMeta      `json:"header"`
	Rows     []renderedRowDTO  `json:"rows"`
	Marked   map[string]string `json:"marked"`   // MarkColor as string; uint64 keys stringified
	Comments map[string]string `json:"comments"` // uint64 keys stringified
	Note     string            `json:"note,omitempty"`
}

type metaOnlyDTO struct {
	Version  int               `json:"version"`
	Marked   map[string]string `json:"marked"`
	Comments map[string]string `json:"comments"`
}

// --- Conversions ---

func toDTORow(r renderedRow) renderedRowDTO {
	return renderedRowDTO{
		Cols:          append([]string(nil), r.cols...),
		Height:        r.height,
		ID:            r.id,
		OriginalIndex: r.originalIndex,
	}
}

func fromDTORow(d renderedRowDTO) renderedRow {
	return renderedRow{
		cols:          append([]string(nil), d.Cols...),
		height:        d.Height,
		id:            d.ID,
		originalIndex: d.OriginalIndex,
	}
}

func u64KeyToStringMarkMap(in map[uint64]MarkColor) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[strconv.FormatUint(k, 10)] = string(v)
	}
	return out
}

func u64KeyToStringStringMap(in map[uint64]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[strconv.FormatUint(k, 10)] = v
	}
	return out
}

func parseUintKeyMapMark(in map[string]string) (map[uint64]MarkColor, error) {
	out := make(map[uint64]MarkColor, len(in))
	for ks, vs := range in {
		k, err := strconv.ParseUint(ks, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid uint64 key %q: %w", ks, err)
		}
		out[k] = sanitizeMarkColor(vs)
	}
	return out, nil
}

func parseUintKeyMapString(in map[string]string) (map[uint64]string, error) {
	out := make(map[uint64]string, len(in))
	for ks, vs := range in {
		k, err := strconv.ParseUint(ks, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid uint64 key %q: %w", ks, err)
		}
		out[k] = vs
	}
	return out, nil
}

// Accept only known values; anything else becomes MarkNone.
func sanitizeMarkColor(s string) MarkColor {
	switch MarkColor(s) {
	case MarkNone, MarkRed, MarkGreen, MarkAmber:
		return MarkColor(s)
	default:
		return MarkNone
	}
}

// --- Public API ---

// ExportModel writes the *currently filtered* rows to a CSV file,
// including mark color and comment as additional columns.
func ExportModel(m *model, path string) error {
	// Open file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("open export file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Build header: original columns + Mark + Comment
	header := make([]string, 0, len(m.data.header)+2)
	for _, col := range m.data.header {
		header = append(header, col.Name)
	}
	header = append(header, "Mark", "Comment")

	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Decide which indices to export:
	// if filteredIndices is empty, fall back to all rows.
	indices := m.data.filteredIndices
	if len(indices) == 0 {
		indices = make([]int, len(m.data.rows))
		for i := range m.data.rows {
			indices[i] = i
		}
	}

	// Export each visible row
	for _, idx := range indices {
		// sanity check
		if idx < 0 || idx >= len(m.data.rows) {
			return fmt.Errorf("filtered index %d out of range", idx)
		}
		r := m.data.rows[idx]

		// row data: original cols
		out := append([]string(nil), r.cols...)

		// append mark + comment using the row's id
		mark := ""
		if c, ok := m.data.markedRows[r.id]; ok {
			mark = string(c)
		}

		comment := ""
		if c, ok := m.data.commentRows[r.id]; ok {
			comment = c
		}

		out = append(out, mark, comment)

		if err := w.Write(out); err != nil {
			return fmt.Errorf("write row %d: %w", idx, err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}

// SaveModel writes the entire model to a JSON file.
func SaveModel(m *model, path string) error {
	dto := snapshotDTO{
		Version:  snapshotVersion,
		Header:   nil, // filled below
		Rows:     make([]renderedRowDTO, 0, len(m.data.rows)),
		Marked:   u64KeyToStringMarkMap(m.data.markedRows),
		Comments: u64KeyToStringStringMap(m.data.commentRows),
	}

	// Copy header metadata
	if len(m.data.header) > 0 {
		dto.Header = make([]ColumnMeta, len(m.data.header))
		copy(dto.Header, m.data.header)
	}

	// Copy rows
	for _, r := range m.data.rows {
		dto.Rows = append(dto.Rows, toDTORow(r))
	}

	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadModel replaces the contents of m with the snapshot from path.
func LoadModel(m *model, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var dto snapshotDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}
	if dto.Version != snapshotVersion {
		return fmt.Errorf("snapshot version %d not supported (want %d)", dto.Version, snapshotVersion)
	}

	// Restore header
	m.data.header = m.data.header[:0]
	if len(dto.Header) > 0 {
		m.data.header = make([]ColumnMeta, len(dto.Header))
		copy(m.data.header, dto.Header)
	}

	// Restore rows
	m.data.rows = m.data.rows[:0]
	for _, dr := range dto.Rows {
		m.data.rows = append(m.data.rows, fromDTORow(dr))
	}

	// Restore marks/comments
	var errMarks, errComments error
	m.data.markedRows, errMarks = parseUintKeyMapMark(dto.Marked)
	if errMarks != nil {
		return errMarks
	}
	m.data.commentRows, errComments = parseUintKeyMapString(dto.Comments)
	if errComments != nil {
		return errComments
	}

	return nil
}

// SaveMeta writes only marks/comments so they can be re-applied after a fresh CSV import.
func SaveMeta(m *model, path string) error {
	dto := metaOnlyDTO{
		Version:  snapshotVersion,
		Marked:   u64KeyToStringMarkMap(m.data.markedRows),
		Comments: u64KeyToStringStringMap(m.data.commentRows),
	}
	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadMeta merges marks/comments into m, only for rows currently present (by ID).
func LoadMeta(m *model, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var dto metaOnlyDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}
	if dto.Version != snapshotVersion {
		return fmt.Errorf("meta version %d not supported (want %d)", dto.Version, snapshotVersion)
	}

	if m.data.markedRows == nil {
		m.data.markedRows = make(map[uint64]MarkColor)
	}
	if m.data.commentRows == nil {
		m.data.commentRows = make(map[uint64]string)
	}

	present := make(map[uint64]struct{}, len(m.data.rows))
	for _, r := range m.data.rows {
		present[r.id] = struct{}{}
	}

	for ks, vs := range dto.Marked {
		k, err := strconv.ParseUint(ks, 10, 64)
		if err != nil {
			return err
		}
		if _, ok := present[k]; ok {
			m.data.markedRows[k] = sanitizeMarkColor(vs)
		}
	}
	for ks, vs := range dto.Comments {
		k, err := strconv.ParseUint(ks, 10, 64)
		if err != nil {
			return err
		}
		if _, ok := present[k]; ok {
			m.data.commentRows[k] = vs
		}
	}

	return nil
}
