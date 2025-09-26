// snapshot.go
package main

import (
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
	Header   renderedRowDTO    `json:"header"`
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

// SaveModel writes the entire model to a JSON file.
func SaveModel(m *model, path string) error {
	dto := snapshotDTO{
		Version:  snapshotVersion,
		Header:   toDTORow(m.header),
		Rows:     make([]renderedRowDTO, 0, len(m.rows)),
		Marked:   u64KeyToStringMarkMap(m.markedRows),
		Comments: u64KeyToStringStringMap(m.commentRows),
	}
	for _, r := range m.rows {
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

	m.header = fromDTORow(dto.Header)

	m.rows = m.rows[:0]
	for _, dr := range dto.Rows {
		m.rows = append(m.rows, fromDTORow(dr))
	}

	if m.markedRows, err = parseUintKeyMapMark(dto.Marked); err != nil {
		return err
	}
	if m.commentRows, err = parseUintKeyMapString(dto.Comments); err != nil {
		return err
	}

	return nil
}

// SaveMeta writes only marks/comments so they can be re-applied after a fresh CSV import.
func SaveMeta(m *model, path string) error {
	dto := metaOnlyDTO{
		Version:  snapshotVersion,
		Marked:   u64KeyToStringMarkMap(m.markedRows),
		Comments: u64KeyToStringStringMap(m.commentRows),
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

	if m.markedRows == nil {
		m.markedRows = make(map[uint64]MarkColor)
	}
	if m.commentRows == nil {
		m.commentRows = make(map[uint64]string)
	}

	present := make(map[uint64]struct{}, len(m.rows))
	for _, r := range m.rows {
		present[r.id] = struct{}{}
	}

	for ks, vs := range dto.Marked {
		k, err := strconv.ParseUint(ks, 10, 64)
		if err != nil {
			return err
		}
		if _, ok := present[k]; ok {
			m.markedRows[k] = sanitizeMarkColor(vs)
		}
	}
	for ks, vs := range dto.Comments {
		k, err := strconv.ParseUint(ks, 10, 64)
		if err != nil {
			return err
		}
		if _, ok := present[k]; ok {
			m.commentRows[k] = vs
		}
	}

	return nil
}
