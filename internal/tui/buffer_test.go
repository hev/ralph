package tui

import (
	"fmt"
	"testing"
)

func TestNewBuffer(t *testing.T) {
	t.Run("default capacity", func(t *testing.T) {
		b := NewBuffer(0)
		if b.Capacity() != DefaultBufferSize {
			t.Errorf("expected capacity %d, got %d", DefaultBufferSize, b.Capacity())
		}
	})

	t.Run("custom capacity", func(t *testing.T) {
		b := NewBuffer(100)
		if b.Capacity() != 100 {
			t.Errorf("expected capacity 100, got %d", b.Capacity())
		}
	})

	t.Run("negative capacity uses default", func(t *testing.T) {
		b := NewBuffer(-5)
		if b.Capacity() != DefaultBufferSize {
			t.Errorf("expected capacity %d, got %d", DefaultBufferSize, b.Capacity())
		}
	})
}

func TestBufferAppend(t *testing.T) {
	t.Run("basic append", func(t *testing.T) {
		b := NewBuffer(10)
		b.Append("line 1", LineTypeDefault)
		b.Append("line 2", LineTypeInfo)

		if b.Size() != 2 {
			t.Errorf("expected size 2, got %d", b.Size())
		}
	})

	t.Run("append wraps around", func(t *testing.T) {
		b := NewBuffer(3)
		b.Append("line 1", LineTypeDefault)
		b.Append("line 2", LineTypeDefault)
		b.Append("line 3", LineTypeDefault)
		b.Append("line 4", LineTypeDefault) // Should overwrite line 1

		if b.Size() != 3 {
			t.Errorf("expected size 3, got %d", b.Size())
		}

		lines := b.GetVisibleLines(3)
		if lines[0].Content != "line 2" {
			t.Errorf("expected oldest line to be 'line 2', got '%s'", lines[0].Content)
		}
		if lines[2].Content != "line 4" {
			t.Errorf("expected newest line to be 'line 4', got '%s'", lines[2].Content)
		}
	})
}

func TestBufferGetVisibleLines(t *testing.T) {
	t.Run("empty buffer", func(t *testing.T) {
		b := NewBuffer(10)
		lines := b.GetVisibleLines(5)
		if lines != nil {
			t.Errorf("expected nil, got %v", lines)
		}
	})

	t.Run("viewport larger than buffer", func(t *testing.T) {
		b := NewBuffer(10)
		b.Append("line 1", LineTypeDefault)
		b.Append("line 2", LineTypeDefault)

		lines := b.GetVisibleLines(5)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}
	})

	t.Run("viewport smaller than buffer", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 5; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		lines := b.GetVisibleLines(3)
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}
		// Should show the most recent 3 lines
		if lines[0].Content != "line 3" {
			t.Errorf("expected 'line 3', got '%s'", lines[0].Content)
		}
		if lines[2].Content != "line 5" {
			t.Errorf("expected 'line 5', got '%s'", lines[2].Content)
		}
	})

	t.Run("preserves line type", func(t *testing.T) {
		b := NewBuffer(10)
		b.Append("error line", LineTypeError)
		b.Append("info line", LineTypeInfo)

		lines := b.GetVisibleLines(2)
		if lines[0].LineType != LineTypeError {
			t.Errorf("expected LineTypeError, got %v", lines[0].LineType)
		}
		if lines[1].LineType != LineTypeInfo {
			t.Errorf("expected LineTypeInfo, got %v", lines[1].LineType)
		}
	})
}

func TestBufferScroll(t *testing.T) {
	t.Run("scroll up and down", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 10; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		// At bottom, should see lines 6-10
		lines := b.GetVisibleLines(5)
		if lines[4].Content != "line 10" {
			t.Errorf("expected 'line 10', got '%s'", lines[4].Content)
		}

		// Scroll up 3 lines, should see lines 3-7
		b.ScrollUp(3)
		lines = b.GetVisibleLines(5)
		if lines[4].Content != "line 7" {
			t.Errorf("expected 'line 7' after scroll up, got '%s'", lines[4].Content)
		}

		// Scroll down 2 lines, should see lines 5-9
		b.ScrollDown(2)
		lines = b.GetVisibleLines(5)
		if lines[4].Content != "line 9" {
			t.Errorf("expected 'line 9' after scroll down, got '%s'", lines[4].Content)
		}
	})

	t.Run("scroll clamps at bottom", func(t *testing.T) {
		b := NewBuffer(10)
		b.Append("line 1", LineTypeDefault)

		b.ScrollDown(100)
		if b.ScrollOffset() != 0 {
			t.Errorf("expected scroll offset 0, got %d", b.ScrollOffset())
		}
	})

	t.Run("scroll to top", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 10; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		b.ScrollToTop()
		lines := b.GetVisibleLines(5)
		if lines[0].Content != "line 1" {
			t.Errorf("expected 'line 1' at top, got '%s'", lines[0].Content)
		}
	})

	t.Run("scroll to bottom", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 10; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		b.ScrollToTop()
		b.ScrollToBottom()

		if !b.IsAtBottom() {
			t.Error("expected to be at bottom")
		}

		lines := b.GetVisibleLines(5)
		if lines[4].Content != "line 10" {
			t.Errorf("expected 'line 10' at bottom, got '%s'", lines[4].Content)
		}
	})
}

func TestBufferPause(t *testing.T) {
	t.Run("paused state", func(t *testing.T) {
		b := NewBuffer(10)
		if b.Paused() {
			t.Error("expected not paused initially")
		}

		b.SetPaused(true)
		if !b.Paused() {
			t.Error("expected paused after SetPaused(true)")
		}

		b.SetPaused(false)
		if b.Paused() {
			t.Error("expected not paused after SetPaused(false)")
		}
	})

	t.Run("toggle paused", func(t *testing.T) {
		b := NewBuffer(10)

		newState := b.TogglePaused()
		if !newState || !b.Paused() {
			t.Error("expected paused after toggle")
		}

		newState = b.TogglePaused()
		if newState || b.Paused() {
			t.Error("expected not paused after second toggle")
		}
	})

	t.Run("paused maintains scroll position on append", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 5; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		b.ScrollUp(2) // Scroll up 2 lines
		b.SetPaused(true)

		// Append a new line while paused
		b.Append("line 6", LineTypeDefault)

		// Scroll offset should be maintained
		if b.ScrollOffset() != 2 {
			t.Errorf("expected scroll offset 2, got %d", b.ScrollOffset())
		}
	})

	t.Run("unpause resets scroll to bottom", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 5; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		b.ScrollUp(2)
		b.SetPaused(true)
		b.SetPaused(false) // Unpause should reset scroll

		if b.ScrollOffset() != 0 {
			t.Errorf("expected scroll offset 0 after unpause, got %d", b.ScrollOffset())
		}
	})

	t.Run("not paused follows new content", func(t *testing.T) {
		b := NewBuffer(10)
		for i := 1; i <= 5; i++ {
			b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
		}

		b.ScrollUp(2) // Scroll up
		// Not paused, so append should reset scroll
		b.Append("line 6", LineTypeDefault)

		if b.ScrollOffset() != 0 {
			t.Errorf("expected scroll offset 0 after append, got %d", b.ScrollOffset())
		}
	})
}

func TestBufferClear(t *testing.T) {
	b := NewBuffer(10)
	for i := 1; i <= 5; i++ {
		b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
	}

	b.ScrollUp(2)
	b.Clear()

	if b.Size() != 0 {
		t.Errorf("expected size 0, got %d", b.Size())
	}
	if b.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0, got %d", b.ScrollOffset())
	}
}

func TestBufferWraparound(t *testing.T) {
	// Test that wraparound works correctly with various operations
	b := NewBuffer(5)

	// Fill buffer
	for i := 1; i <= 5; i++ {
		b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
	}

	// Wrap around multiple times
	for i := 6; i <= 15; i++ {
		b.Append(fmt.Sprintf("line %d", i), LineTypeDefault)
	}

	// Should have lines 11-15
	lines := b.GetVisibleLines(5)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}

	expected := []string{"line 11", "line 12", "line 13", "line 14", "line 15"}
	for i, exp := range expected {
		if lines[i].Content != exp {
			t.Errorf("at index %d: expected '%s', got '%s'", i, exp, lines[i].Content)
		}
	}
}
