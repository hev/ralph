package tui

import (
	"io"
	"os"
)

// Key represents a keyboard input event
type Key int

const (
	KeyNone Key = iota
	KeySpace
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyJ
	KeyK
	KeyG
	KeyShiftG
	KeyQ
	KeyEscape
	KeyCtrlC
)

// InputReader reads keyboard input from the terminal in raw mode.
// It parses escape sequences for special keys like arrows and page up/down.
type InputReader struct {
	input  io.Reader
	buf    []byte
	closed bool
}

// NewInputReader creates a new InputReader that reads from stdin.
func NewInputReader() *InputReader {
	return &InputReader{
		input: os.Stdin,
		buf:   make([]byte, 16), // Buffer for escape sequences
	}
}

// NewInputReaderFromFd creates a new InputReader from a file descriptor.
func NewInputReaderFromFd(fd int) *InputReader {
	return &InputReader{
		input: os.NewFile(uintptr(fd), "stdin"),
		buf:   make([]byte, 16),
	}
}

// ReadKey blocks until a key is pressed and returns the parsed Key.
// Returns KeyNone and an error on read failure.
func (r *InputReader) ReadKey() (Key, error) {
	if r.closed {
		return KeyNone, io.EOF
	}

	// Read first byte
	n, err := r.input.Read(r.buf[:1])
	if err != nil {
		return KeyNone, err
	}
	if n == 0 {
		return KeyNone, nil
	}

	b := r.buf[0]

	// Check for Ctrl+C (ETX)
	if b == 0x03 {
		return KeyCtrlC, nil
	}

	// Check for Escape
	if b == 0x1B {
		return r.parseEscapeSequence()
	}

	// Space
	if b == ' ' {
		return KeySpace, nil
	}

	// Single character keys
	switch b {
	case 'j':
		return KeyJ, nil
	case 'k':
		return KeyK, nil
	case 'g':
		return KeyG, nil
	case 'G':
		return KeyShiftG, nil
	case 'q':
		return KeyQ, nil
	}

	return KeyNone, nil
}

// parseEscapeSequence handles escape sequences after reading 0x1B.
// Common sequences:
//   - ESC [ A = Up
//   - ESC [ B = Down
//   - ESC [ C = Right
//   - ESC [ D = Left
//   - ESC [ 5 ~ = Page Up
//   - ESC [ 6 ~ = Page Down
//   - ESC [ H = Home
//   - ESC [ F = End
func (r *InputReader) parseEscapeSequence() (Key, error) {
	// Try to read the next byte with a short timeout approach
	// In raw mode, we should get the full sequence quickly
	n, err := r.input.Read(r.buf[:1])
	if err != nil {
		return KeyEscape, nil // Just Escape key
	}
	if n == 0 {
		return KeyEscape, nil
	}

	// Check for CSI (Control Sequence Introducer)
	if r.buf[0] != '[' {
		return KeyEscape, nil
	}

	// Read the next character(s)
	n, err = r.input.Read(r.buf[:1])
	if err != nil || n == 0 {
		return KeyEscape, nil
	}

	switch r.buf[0] {
	case 'A':
		return KeyUp, nil
	case 'B':
		return KeyDown, nil
	case 'C':
		return KeyRight, nil
	case 'D':
		return KeyLeft, nil
	case 'H':
		return KeyHome, nil
	case 'F':
		return KeyEnd, nil
	case '5':
		// Page Up: ESC [ 5 ~
		r.input.Read(r.buf[:1]) // Read the trailing ~
		return KeyPageUp, nil
	case '6':
		// Page Down: ESC [ 6 ~
		r.input.Read(r.buf[:1]) // Read the trailing ~
		return KeyPageDown, nil
	}

	return KeyNone, nil
}

// Close marks the reader as closed. Future reads will return io.EOF.
func (r *InputReader) Close() {
	r.closed = true
}

// String returns the display name of a key
func (k Key) String() string {
	switch k {
	case KeySpace:
		return "Space"
	case KeyUp:
		return "Up"
	case KeyDown:
		return "Down"
	case KeyLeft:
		return "Left"
	case KeyRight:
		return "Right"
	case KeyPageUp:
		return "PageUp"
	case KeyPageDown:
		return "PageDown"
	case KeyHome:
		return "Home"
	case KeyEnd:
		return "End"
	case KeyJ:
		return "j"
	case KeyK:
		return "k"
	case KeyG:
		return "g"
	case KeyShiftG:
		return "G"
	case KeyQ:
		return "q"
	case KeyEscape:
		return "Escape"
	case KeyCtrlC:
		return "Ctrl+C"
	default:
		return "None"
	}
}
