package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type FooterState struct {
	Mode      Command
	ModeInput string

	FileName string

	FilterLabel string
	MarksOnly   bool

	Row       int
	TotalRows int

	StatusMessage string
	Legend        string
}

type FooterStyles struct {
	BarBG      lipgloss.Color
	StatusBG   lipgloss.Color
	ModePillBG lipgloss.Color
	ModePillFG lipgloss.Color
	FileNameFG lipgloss.Color
	TextFG     lipgloss.Color
	DimFG      lipgloss.Color
	StatusFG   lipgloss.Color
	LegendFG   lipgloss.Color
}

func DefaultFooterStyles() FooterStyles {
	return FooterStyles{
		BarBG:      lipgloss.Color("#2b2b2b"),
		StatusBG:   lipgloss.Color("#000000"),
		ModePillBG: lipgloss.Color("#ff9f1c"),
		ModePillFG: lipgloss.Color("#000000"),
		FileNameFG: lipgloss.Color("#e0e0e0"),
		TextFG:     lipgloss.Color("#cfcfcf"),
		DimFG:      lipgloss.Color("#a0a0a0"),
		StatusFG:   lipgloss.Color("#9a9a9a"),
		LegendFG:   lipgloss.Color("#b0b0b0"),
	}
}

func RenderFooter(width int, st FooterState, styles FooterStyles) string {
	if width <= 0 {
		return ""
	}
	if st.FilterLabel == "" {
		st.FilterLabel = "None"
	}
	if st.Legend == "" {
		st.Legend = "(? help · f filter · c comment)"
	}
	if st.Row < 0 {
		st.Row = 0
	}
	if st.TotalRows < 0 {
		st.TotalRows = 0
	}

	line1 := renderControlBar(width, st, styles)
	line2 := renderStatusBar(width, st, styles)
	return line1 + "\n" + line2
}

func renderControlBar(width int, st FooterState, styles FooterStyles) string {
	gapW := 1
	filterValW := 12
	marksW := 5
	statusFixedW := runeWidth(fmt.Sprintf("[FILTER: %s] · [MARKS ONLY: %s]", strings.Repeat("X", filterValW), strings.Repeat("X", marksW)))

	rightPlain := fmt.Sprintf(" Rows %d/%d", st.Row, st.TotalRows)
	rightPlain = truncatePlain(rightPlain, width)
	rightW := runeWidth(rightPlain)

	leftW := width - rightW
	if leftW < 0 {
		leftW = 0
	}

	modeColW := clamp(leftW/4, 20, 36)
	statusColW := statusFixedW
	fileColW := leftW - modeColW - statusColW - 2*gapW
	if fileColW < 0 {
		deficit := -fileColW
		if statusColW > 10 {
			shrink := min(deficit, statusColW-10)
			statusColW -= shrink
			deficit -= shrink
		}
		if deficit > 0 && modeColW > 10 {
			shrink := min(deficit, modeColW-10)
			modeColW -= shrink
			deficit -= shrink
		}
		fileColW = leftW - modeColW - statusColW - 2*gapW
		if fileColW < 0 {
			modeColW = max(0, modeColW+fileColW)
			fileColW = 0
		}
	}

	modeText := commandLabel(st.Mode)
	innerModeW := max(0, modeColW-2)
	modePillW := modeColW
	if runeWidth(modeText) <= innerModeW {
		modePillW = runeWidth(modeText) + 2
	}
	modeSlack := modeColW - modePillW
	if modeSlack > 0 {
		modeColW = modePillW
		fileColW += modeSlack
	}

	modeSeg := renderModeSegment(modeColW, st, styles)
	fileSeg := renderFileSegment(fileColW, st, styles)
	statusSeg := renderFilterMarksSegment(statusColW, st, styles, filterValW, marksW)

	left := modeSeg + strings.Repeat(" ", gapW) + fileSeg + strings.Repeat(" ", gapW) + statusSeg
	leftWActual := modeColW + fileColW + statusColW + 2*gapW
	if leftWActual < leftW {
		left += strings.Repeat(" ", leftW-leftWActual)
	}

	linePlain := left + rightPlain
	return applyBar(linePlain, styles.BarBG, styles.TextFG)
}

func renderStatusBar(width int, st FooterState, styles FooterStyles) string {
	legendPlain := truncatePlain(st.Legend, width)
	legendW := runeWidth(legendPlain)

	leftW := width - legendW
	if leftW < 0 {
		leftW = 0
	}

	msgPlain := truncatePlain(st.StatusMessage, leftW)
	msgPlain = padRightPlain(msgPlain, leftW)

	linePlain := applyFG(msgPlain, styles.StatusFG, styles.StatusFG) + applyFG(legendPlain, styles.LegendFG, styles.StatusFG)
	return applyBar(linePlain, styles.StatusBG, styles.StatusFG)
}

func renderModeSegment(colW int, st FooterState, styles FooterStyles) string {
	if colW <= 0 {
		return ""
	}
	content := commandLabel(st.Mode)
	innerW := max(0, colW-2)
	content = truncatePlain(content, innerW)
	pillPlain := " " + content + " "
	pillPlain = truncatePlain(pillPlain, colW)
	pad := strings.Repeat(" ", colW-runeWidth(pillPlain))

	pill := ansiBg(styles.ModePillBG) + ansiFg(styles.ModePillFG) + pillPlain
	pill += ansiBg(styles.BarBG) + ansiFg(styles.TextFG) + pad
	return pill
}

func renderFileSegment(colW int, st FooterState, styles FooterStyles) string {
	if colW <= 0 {
		return ""
	}
	name := strings.TrimSpace(st.FileName)
	if name == "" {
		name = "(no file)"
	}
	innerW := max(0, colW-2)
	inner := truncatePlain(name, innerW)
	filePlain := inner
	remaining := colW
	prefix := "▸ "
	mid := " ▸ "
	inputPlain := ""
	if remaining > 0 {
		filePlain = truncatePlain(prefix+filePlain, remaining)
		remaining -= runeWidth(filePlain)
	}
	if remaining > 0 {
		input := strings.TrimSpace(st.ModeInput)
		if input != "" {
			inputPlain = mid + input
			inputPlain = truncatePlain(inputPlain, remaining)
			remaining -= runeWidth(inputPlain)
		}
	}
	if remaining < 0 {
		remaining = 0
	}

	pad := strings.Repeat(" ", remaining)
	return applyFG(filePlain, styles.FileNameFG, styles.TextFG) + inputPlain + pad
}

func renderFilterMarksSegment(colW int, st FooterState, styles FooterStyles, filterValW, marksW int) string {
	if colW <= 0 {
		return ""
	}
	filterVal := truncatePlain(strings.TrimSpace(st.FilterLabel), filterValW)
	marks := truncatePlain(fmt.Sprintf("%v", st.MarksOnly), marksW)

	plain := fmt.Sprintf("[FILTER: %s] · [MARKS ONLY: %s]", filterVal, marks)
	plain = truncatePlain(plain, colW)
	plain = padRightPlain(plain, colW)
	return applyFG(plain, styles.DimFG, styles.TextFG)
}

func applyBar(s string, bg lipgloss.Color, baseFG lipgloss.Color) string {
	return ansiBg(bg) + ansiFg(baseFG) + s + "\x1b[0m"
}

func commandLabel(cmd Command) string {
	switch cmd {
	case CmdJump:
		return "JUMP"
	case CmdSearch:
		return "SEARCH"
	case CmdFilter:
		return "FILTER"
	case CmdComment:
		return "COMMENT"
	case CmdMark:
		return "MARK"
	default:
		return "NORMAL"
	}
}

func applyFG(s string, fg lipgloss.Color, resetFG lipgloss.Color) string {
	return ansiFg(fg) + s + ansiFg(resetFG)
}

func ansiFg(c lipgloss.Color) string {
	return ansiColor(false, c)
}

func ansiBg(c lipgloss.Color) string {
	return ansiColor(true, c)
}

func ansiColor(isBg bool, c lipgloss.Color) string {
	s := string(c)
	if s == "" {
		if isBg {
			return "\x1b[49m"
		}
		return "\x1b[39m"
	}
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		r, _ := strconv.ParseInt(s[1:3], 16, 0)
		g, _ := strconv.ParseInt(s[3:5], 16, 0)
		b, _ := strconv.ParseInt(s[5:7], 16, 0)
		code := 38
		if isBg {
			code = 48
		}
		return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", code, r, g, b)
	}
	return ""
}

func padRightPlain(s string, w int) string {
	if w <= 0 {
		return ""
	}
	cur := runeWidth(s)
	if cur >= w {
		return s
	}
	return s + strings.Repeat(" ", w-cur)
}

func truncatePlain(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w])
}

func runeWidth(s string) int {
	return len([]rune(s))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
