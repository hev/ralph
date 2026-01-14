package tui

import (
	"fmt"
	"io"
	"strings"
)

// FooterHeight is the number of lines reserved for the footer
const FooterHeight = 4

// ANSI escape sequences for cursor control and screen management
const (
	escClearScreen     = "\033[2J"
	escClearLine       = "\033[2K"
	escCursorHome      = "\033[H"
	escCursorHide      = "\033[?25l"
	escCursorShow      = "\033[?25h"
	escSaveCursor      = "\033[s"
	escRestoreCursor   = "\033[u"
	escScrollRegionFmt = "\033[%d;%dr" // top;bottom
	escResetScroll     = "\033[r"
	escCursorPosFmt    = "\033[%d;%dH" // row;col (1-indexed)
)

// Renderer handles drawing the TUI to the terminal
type Renderer struct {
	out    io.Writer
	width  int
	height int
}

// NewRenderer creates a new Renderer writing to the given output
func NewRenderer(out io.Writer) *Renderer {
	return &Renderer{
		out: out,
	}
}

// SetSize updates the terminal dimensions
func (r *Renderer) SetSize(width, height int) {
	r.width = width
	r.height = height
}

// Width returns the current terminal width
func (r *Renderer) Width() int {
	return r.width
}

// Height returns the current terminal height
func (r *Renderer) Height() int {
	return r.height
}

// LogAreaHeight returns the height available for log lines
func (r *Renderer) LogAreaHeight() int {
	h := r.height - FooterHeight
	if h < 1 {
		return 1
	}
	return h
}

// Init initializes the terminal for TUI rendering
func (r *Renderer) Init() {
	// Hide cursor, clear screen
	fmt.Fprint(r.out, escCursorHide)
	fmt.Fprint(r.out, escClearScreen)
	fmt.Fprint(r.out, escCursorHome)
}

// Cleanup restores terminal to normal state
func (r *Renderer) Cleanup() {
	// Reset scroll region, show cursor, move to bottom
	fmt.Fprint(r.out, escResetScroll)
	fmt.Fprint(r.out, escCursorShow)
	fmt.Fprintf(r.out, escCursorPosFmt, r.height, 1)
	fmt.Fprintln(r.out) // Leave cursor on new line
}

// SetScrollRegion sets the terminal scroll region to the log area
func (r *Renderer) SetScrollRegion() {
	logHeight := r.LogAreaHeight()
	if logHeight > 0 {
		fmt.Fprintf(r.out, escScrollRegionFmt, 1, logHeight)
	}
}

// Draw renders the complete UI: log area + footer
func (r *Renderer) Draw(buffer *Buffer, state StateSnapshot) {
	if r.width == 0 || r.height == 0 {
		return
	}

	// Draw log lines
	r.drawLogArea(buffer)

	// Draw footer (always at bottom)
	r.drawFooter(state)
}

// drawLogArea renders the scrollable log area
func (r *Renderer) drawLogArea(buffer *Buffer) {
	logHeight := r.LogAreaHeight()
	lines := buffer.GetVisibleLines(logHeight)

	// Position cursor at top of log area
	fmt.Fprintf(r.out, escCursorPosFmt, 1, 1)

	for i := 0; i < logHeight; i++ {
		fmt.Fprint(r.out, escClearLine)
		if i < len(lines) {
			line := lines[i]
			style := StyleForLineType(line.LineType)
			content := r.truncate(line.Content, r.width)
			fmt.Fprint(r.out, style.Render(content))
		}
		if i < logHeight-1 {
			fmt.Fprintln(r.out)
		}
	}
}

// drawFooter renders the pinned 4-line footer at the bottom
func (r *Renderer) drawFooter(state StateSnapshot) {
	footerStart := r.height - FooterHeight + 1
	if footerStart < 1 {
		footerStart = 1
	}

	// Line 1: Phase + pause indicator
	fmt.Fprintf(r.out, escCursorPosFmt, footerStart, 1)
	fmt.Fprint(r.out, escClearLine)
	r.drawPhaseLine(state)

	// Line 2: Todo progress bar
	fmt.Fprintf(r.out, escCursorPosFmt, footerStart+1, 1)
	fmt.Fprint(r.out, escClearLine)
	r.drawTodoLine(state)

	// Line 3: Current todo text
	fmt.Fprintf(r.out, escCursorPosFmt, footerStart+2, 1)
	fmt.Fprint(r.out, escClearLine)
	r.drawCurrentTodoLine(state)

	// Line 4: Token utilization
	fmt.Fprintf(r.out, escCursorPosFmt, footerStart+3, 1)
	fmt.Fprint(r.out, escClearLine)
	r.drawTokenLine(state)
}

// drawPhaseLine renders: "  Main Loop (sonnet)  [PAUSED]"
func (r *Renderer) drawPhaseLine(state StateSnapshot) {
	phaseDisplay := state.Phase.String()
	if state.Model != "" {
		phaseDisplay = fmt.Sprintf("%s (%s)", phaseDisplay, state.Model)
	}
	phaseText := FooterPhase.Render("  " + phaseDisplay)

	var pauseText string
	if state.Paused {
		pauseText = "  " + FooterPaused.Render("[PAUSED]")
	}

	fmt.Fprint(r.out, phaseText+pauseText)
}

// drawTodoLine renders: "  Todos: ████████░░░░░░░░ 5/12 (42%)"
func (r *Renderer) drawTodoLine(state StateSnapshot) {
	label := FooterLabel.Render("  Todos: ")

	if state.TodosTotal == 0 {
		fmt.Fprint(r.out, label+FooterValue.Render("none"))
		return
	}

	bar := r.renderProgressBar(state.TodosCompleted, state.TodosTotal, 16, false)
	pct := 0
	if state.TodosTotal > 0 {
		pct = state.TodosCompleted * 100 / state.TodosTotal
	}
	value := FooterValue.Render(fmt.Sprintf(" %d/%d (%d%%)", state.TodosCompleted, state.TodosTotal, pct))

	fmt.Fprint(r.out, label+bar+value)
}

// drawCurrentTodoLine renders: "  Working on: Fix authentication bug"
func (r *Renderer) drawCurrentTodoLine(state StateSnapshot) {
	label := FooterLabel.Render("  Working on: ")

	if state.CurrentTodo == "" {
		fmt.Fprint(r.out, label+FooterValue.Render("nothing"))
		return
	}

	// Truncate todo text to fit
	maxLen := r.width - 16 // "  Working on: " is ~14 chars
	if maxLen < 10 {
		maxLen = 10
	}
	todoText := r.truncate(state.CurrentTodo, maxLen)
	fmt.Fprint(r.out, label+FooterTodo.Render(todoText))
}

// drawTokenLine renders: "  Context: ██████░░░░░░░░░ 45k/200k"
func (r *Renderer) drawTokenLine(state StateSnapshot) {
	label := FooterLabel.Render("  Context: ")

	bar := r.renderProgressBar(state.TokensUsed, state.TokensMax, 16, true)
	value := FooterValue.Render(fmt.Sprintf(" %s/%s", formatTokenCount(state.TokensUsed), formatTokenCount(state.TokensMax)))

	fmt.Fprint(r.out, label+bar+value)
}

// renderProgressBar creates a progress bar string
// useWarningColors enables yellow/red coloring based on percentage
func (r *Renderer) renderProgressBar(current, max, width int, useWarningColors bool) string {
	if max <= 0 {
		return ProgressBarEmpty.Render(strings.Repeat(ProgressEmpty, width))
	}

	pct := float64(current) / float64(max)
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}

	filled := int(pct * float64(width))
	empty := width - filled

	var filledStyle = ProgressBarFilled
	if useWarningColors {
		if pct > 0.9 {
			filledStyle = ProgressBarCritical
		} else if pct > 0.75 {
			filledStyle = ProgressBarWarning
		}
	}

	filledStr := filledStyle.Render(strings.Repeat(ProgressFilled, filled))
	emptyStr := ProgressBarEmpty.Render(strings.Repeat(ProgressEmpty, empty))

	return filledStr + emptyStr
}

// truncate shortens a string to fit within maxLen, adding "..." if needed
func (r *Renderer) truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatTokenCount formats token counts for display (e.g., 45000 -> "45k")
func formatTokenCount(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 10000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%dk", n/1000)
}

// WritePlain writes a line directly without TUI formatting (for non-TTY mode)
func (r *Renderer) WritePlain(line Line) {
	fmt.Fprintln(r.out, line.Content)
}
