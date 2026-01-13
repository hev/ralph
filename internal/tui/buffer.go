package tui

import "sync"

// DefaultBufferSize is the default number of lines to keep in the buffer
const DefaultBufferSize = 10000

// Line represents a single line in the buffer with its style
type Line struct {
	Content  string
	LineType LineType
}

// Buffer is a thread-safe ring buffer for storing log lines
// with support for scrolling and pause functionality.
type Buffer struct {
	mu sync.RWMutex

	lines    []Line
	capacity int
	head     int  // Index where next write goes
	size     int  // Current number of lines in buffer
	wrapped  bool // Whether buffer has wrapped around

	// Scroll state
	scrollOffset int  // Lines scrolled up from bottom (0 = at bottom)
	paused       bool // When paused, new lines don't auto-scroll
}

// NewBuffer creates a new Buffer with the specified capacity
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = DefaultBufferSize
	}
	return &Buffer{
		lines:    make([]Line, capacity),
		capacity: capacity,
	}
}

// Append adds a new line to the buffer
func (b *Buffer) Append(content string, lineType LineType) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines[b.head] = Line{
		Content:  content,
		LineType: lineType,
	}

	b.head = (b.head + 1) % b.capacity
	if b.size < b.capacity {
		b.size++
	} else {
		b.wrapped = true
	}

	// If not paused, reset scroll to follow new content
	if !b.paused {
		b.scrollOffset = 0
	}
}

// GetVisibleLines returns the lines that should be displayed
// given the current scroll position and viewport height.
// Returns lines in display order (oldest first).
func (b *Buffer) GetVisibleLines(viewportHeight int) []Line {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.size == 0 || viewportHeight <= 0 {
		return nil
	}

	// Calculate how many lines we can show
	numLines := viewportHeight
	if numLines > b.size {
		numLines = b.size
	}

	// Calculate the starting position accounting for scroll
	// scrollOffset=0 means we're at the bottom (most recent)
	// scrollOffset=N means we're showing N lines older

	// Maximum scroll is to the oldest line
	maxScroll := b.size - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Clamp scroll offset
	effectiveScroll := b.scrollOffset
	if effectiveScroll > maxScroll {
		effectiveScroll = maxScroll
	}

	// Calculate the index of the oldest visible line
	// In a ring buffer:
	// - head points to where next write goes
	// - oldest line is at (head - size) % capacity when wrapped
	// - newest line is at (head - 1) % capacity

	result := make([]Line, numLines)

	// Calculate start position:
	// We want to show lines ending at (head - 1 - scrollOffset)
	// and starting at (head - 1 - scrollOffset - numLines + 1)

	endIdx := (b.head - 1 - effectiveScroll + b.capacity) % b.capacity
	startIdx := (endIdx - numLines + 1 + b.capacity) % b.capacity

	// Copy lines in order
	for i := 0; i < numLines; i++ {
		idx := (startIdx + i) % b.capacity
		result[i] = b.lines[idx]
	}

	return result
}

// Size returns the current number of lines in the buffer
func (b *Buffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

// Capacity returns the maximum capacity of the buffer
func (b *Buffer) Capacity() int {
	return b.capacity
}

// ScrollUp scrolls up by n lines (toward older content)
func (b *Buffer) ScrollUp(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.scrollOffset += n
	// Clamp will happen in GetVisibleLines
}

// ScrollDown scrolls down by n lines (toward newer content)
func (b *Buffer) ScrollDown(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.scrollOffset -= n
	if b.scrollOffset < 0 {
		b.scrollOffset = 0
	}
}

// ScrollToTop jumps to the oldest line in the buffer
func (b *Buffer) ScrollToTop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set scroll to maximum (will be clamped by GetVisibleLines)
	b.scrollOffset = b.size
}

// ScrollToBottom jumps to the newest line in the buffer
func (b *Buffer) ScrollToBottom() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.scrollOffset = 0
}

// ScrollOffset returns the current scroll offset
func (b *Buffer) ScrollOffset() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.scrollOffset
}

// Paused returns whether auto-scroll is paused
func (b *Buffer) Paused() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.paused
}

// SetPaused sets the paused state
func (b *Buffer) SetPaused(paused bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.paused = paused
	if !paused {
		b.scrollOffset = 0 // Resume at bottom
	}
}

// TogglePaused toggles the paused state and returns the new state
func (b *Buffer) TogglePaused() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.paused = !b.paused
	if !b.paused {
		b.scrollOffset = 0 // Resume at bottom
	}
	return b.paused
}

// Clear empties the buffer
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.head = 0
	b.size = 0
	b.wrapped = false
	b.scrollOffset = 0
}

// IsAtBottom returns true if viewing the most recent content
func (b *Buffer) IsAtBottom() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.scrollOffset == 0
}
