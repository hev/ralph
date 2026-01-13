package tui

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func setNoColorProfile(t *testing.T) {
	t.Helper()

	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(prev)
	})
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inEsc {
			if c >= '@' && c <= '~' {
				inEsc = false
			}
			continue
		}
		if c == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(c)
	}

	return b.String()
}

func countRune(s string, target rune) int {
	count := 0
	for _, r := range s {
		if r == target {
			count++
		}
	}
	return count
}

func TestRendererRenderProgressBarCounts(t *testing.T) {
	setNoColorProfile(t)

	r := &Renderer{}
	filledRune := []rune(ProgressFilled)[0]
	emptyRune := []rune(ProgressEmpty)[0]

	tests := []struct {
		name          string
		current       int
		max           int
		width         int
		expectedFill  int
		expectedEmpty int
	}{
		{
			name:          "empty max renders all empty",
			current:       0,
			max:           0,
			width:         16,
			expectedFill:  0,
			expectedEmpty: 16,
		},
		{
			name:          "half full bar",
			current:       5,
			max:           10,
			width:         16,
			expectedFill:  8,
			expectedEmpty: 8,
		},
		{
			name:          "over max clamps to full",
			current:       12,
			max:           10,
			width:         16,
			expectedFill:  16,
			expectedEmpty: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := stripANSI(r.renderProgressBar(tt.current, tt.max, tt.width, false))
			if got := utf8.RuneCountInString(bar); got != tt.width {
				t.Fatalf("expected %d runes, got %d (%q)", tt.width, got, bar)
			}
			if got := countRune(bar, filledRune); got != tt.expectedFill {
				t.Fatalf("expected %d filled runes, got %d (%q)", tt.expectedFill, got, bar)
			}
			if got := countRune(bar, emptyRune); got != tt.expectedEmpty {
				t.Fatalf("expected %d empty runes, got %d (%q)", tt.expectedEmpty, got, bar)
			}
		})
	}
}

func TestRendererDrawFooterOutput(t *testing.T) {
	setNoColorProfile(t)

	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.SetSize(80, 10)

	state := StateSnapshot{
		Phase:          PhaseMainLoop,
		TodosCompleted: 5,
		TodosTotal:     12,
		CurrentTodo:    "Fix authentication bug",
		TokensUsed:     45000,
		TokensMax:      200000,
		Paused:         true,
	}

	r.drawFooter(state)

	output := stripANSI(buf.String())
	if !strings.Contains(output, "Main Loop") {
		t.Fatalf("expected phase in output, got %q", output)
	}
	if !strings.Contains(output, "[PAUSED]") {
		t.Fatalf("expected pause indicator in output, got %q", output)
	}
	if !strings.Contains(output, "Todos:") {
		t.Fatalf("expected todo label in output, got %q", output)
	}
	if !strings.Contains(output, "5/12 (41%)") {
		t.Fatalf("expected todo counts in output, got %q", output)
	}
	if !strings.Contains(output, "Working on: Fix authentication bug") {
		t.Fatalf("expected current todo in output, got %q", output)
	}
	if !strings.Contains(output, "Context:") {
		t.Fatalf("expected context label in output, got %q", output)
	}
	if !strings.Contains(output, "45k/200k") {
		t.Fatalf("expected token counts in output, got %q", output)
	}

	expectedTodoBar := stripANSI(r.renderProgressBar(5, 12, 16, false))
	if !strings.Contains(output, expectedTodoBar) {
		t.Fatalf("expected todo progress bar in output, got %q", output)
	}

	expectedTokenBar := stripANSI(r.renderProgressBar(45000, 200000, 16, true))
	if !strings.Contains(output, expectedTokenBar) {
		t.Fatalf("expected token progress bar in output, got %q", output)
	}
}

func TestRendererTruncate(t *testing.T) {
	r := &Renderer{}

	if got := r.truncate("0123456789", 6); got != "012..." {
		t.Fatalf("expected truncated string, got %q", got)
	}
	if got := r.truncate("abc", 3); got != "abc" {
		t.Fatalf("expected full string, got %q", got)
	}
	if got := r.truncate("abcdef", 2); got != "ab" {
		t.Fatalf("expected shortened string, got %q", got)
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{name: "under 1000", input: 999, want: "999"},
		{name: "under 10000", input: 1500, want: "1.5k"},
		{name: "ten thousand", input: 10000, want: "10k"},
		{name: "large", input: 45000, want: "45k"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTokenCount(tt.input); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
