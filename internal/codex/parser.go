package codex

import (
	"fmt"

	"github.com/hev/ralph/internal/claude"
)

// ParseAndPrint outputs a line from Codex without structured parsing.
func ParseAndPrint(line string) bool {
	if line == "" {
		return false
	}
	fmt.Println(line)
	return true
}

// ParseAndPrintTo outputs a line from Codex via the writer callback.
// This allows routing output to TUI or other destinations.
// Codex output is treated as default line type since it's not structured JSON.
func ParseAndPrintTo(line string, writer claude.LineWriter) bool {
	if line == "" {
		return false
	}
	writer(line, claude.TUILineTypeDefault)
	return true
}
