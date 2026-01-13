package tui

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// Terminal manages raw mode and terminal size for the TUI.
type Terminal struct {
	mu sync.RWMutex

	// File descriptors
	stdin  int
	stdout int

	// Terminal state
	isTerminal  bool
	width       int
	height      int
	oldState    *term.State
	inRawMode   bool
	resizeChan  chan struct{}
	resizeStop  chan struct{}
	resizeDone  chan struct{}
}

// NewTerminal creates a Terminal from stdin/stdout file descriptors.
func NewTerminal() *Terminal {
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())

	t := &Terminal{
		stdin:      stdinFd,
		stdout:     stdoutFd,
		isTerminal: term.IsTerminal(stdinFd) && term.IsTerminal(stdoutFd),
	}

	// Get initial size if we're a terminal
	if t.isTerminal {
		w, h, err := term.GetSize(stdoutFd)
		if err == nil {
			t.width = w
			t.height = h
		}
	}

	return t
}

// IsTerminal returns whether stdin/stdout are connected to a TTY.
func (t *Terminal) IsTerminal() bool {
	return t.isTerminal
}

// Size returns the current terminal width and height.
func (t *Terminal) Size() (width, height int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.width, t.height
}

// updateSize refreshes the cached terminal size.
func (t *Terminal) updateSize() {
	if !t.isTerminal {
		return
	}

	w, h, err := term.GetSize(t.stdout)
	if err == nil {
		t.mu.Lock()
		t.width = w
		t.height = h
		t.mu.Unlock()
	}
}

// EnterRawMode puts the terminal into raw mode for direct input handling.
// Returns an error if not a terminal or if entering raw mode fails.
func (t *Terminal) EnterRawMode() error {
	if !t.isTerminal {
		return nil // No-op for non-terminals
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.inRawMode {
		return nil // Already in raw mode
	}

	oldState, err := term.MakeRaw(t.stdin)
	if err != nil {
		return err
	}

	t.oldState = oldState
	t.inRawMode = true
	return nil
}

// Restore returns the terminal to its original state.
// Safe to call multiple times.
func (t *Terminal) Restore() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.inRawMode || t.oldState == nil {
		return nil // Not in raw mode
	}

	err := term.Restore(t.stdin, t.oldState)
	if err == nil {
		t.inRawMode = false
		t.oldState = nil
	}
	return err
}

// InRawMode returns whether the terminal is currently in raw mode.
func (t *Terminal) InRawMode() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.inRawMode
}

// WatchResize starts a goroutine that watches for terminal resize events
// (SIGWINCH) and sends on the returned channel when the size changes.
// Call StopWatchResize to stop watching.
func (t *Terminal) WatchResize() <-chan struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If already watching, return existing channel
	if t.resizeChan != nil {
		return t.resizeChan
	}

	t.resizeChan = make(chan struct{}, 1) // Buffered to avoid blocking signal handler
	t.resizeStop = make(chan struct{})
	t.resizeDone = make(chan struct{})

	// Set up signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		defer close(t.resizeDone)
		defer signal.Stop(sigChan)

		for {
			select {
			case <-sigChan:
				t.updateSize()
				// Non-blocking send
				select {
				case t.resizeChan <- struct{}{}:
				default:
				}
			case <-t.resizeStop:
				return
			}
		}
	}()

	return t.resizeChan
}

// StopWatchResize stops watching for resize events.
// Safe to call if not watching.
func (t *Terminal) StopWatchResize() {
	t.mu.Lock()
	stop := t.resizeStop
	done := t.resizeDone
	t.mu.Unlock()

	if stop == nil {
		return // Not watching
	}

	close(stop)
	<-done // Wait for goroutine to finish

	t.mu.Lock()
	t.resizeChan = nil
	t.resizeStop = nil
	t.resizeDone = nil
	t.mu.Unlock()
}

// StdinFd returns the stdin file descriptor for reading input.
func (t *Terminal) StdinFd() int {
	return t.stdin
}

// StdoutFd returns the stdout file descriptor for writing output.
func (t *Terminal) StdoutFd() int {
	return t.stdout
}
