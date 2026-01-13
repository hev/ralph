package tui

import (
	"context"
	"os"
	"sync"
	"time"
)

// TUI is the main terminal UI coordinator that manages the terminal,
// buffer, renderer, and input handling.
type TUI struct {
	terminal *Terminal
	buffer   *Buffer
	renderer *Renderer
	state    *UIState
	input    *InputReader

	// Configuration
	bufferSize int
	enabled    bool

	// Lifecycle management
	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	doneChan chan struct{}

	// Render ticker for periodic updates
	renderInterval time.Duration
}

// Option is a functional option for configuring the TUI
type Option func(*TUI)

// WithBufferSize sets the log buffer size
func WithBufferSize(size int) Option {
	return func(t *TUI) {
		t.bufferSize = size
	}
}

// WithRenderInterval sets how often the UI redraws (default 16ms ~60fps)
func WithRenderInterval(d time.Duration) Option {
	return func(t *TUI) {
		t.renderInterval = d
	}
}

// WithEnabled toggles whether the TUI renders in interactive mode.
func WithEnabled(enabled bool) Option {
	return func(t *TUI) {
		t.enabled = enabled
	}
}

// New creates a new TUI with the given options.
// The TUI is not started until Start() is called.
func New(opts ...Option) *TUI {
	t := &TUI{
		bufferSize:     DefaultBufferSize,
		renderInterval: 16 * time.Millisecond, // ~60fps
		enabled:        true,
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	// Initialize components
	t.terminal = NewTerminal()
	if !t.terminal.IsTerminal() {
		t.enabled = false
	}
	t.buffer = NewBuffer(t.bufferSize)
	t.renderer = NewRenderer(os.Stdout)
	t.state = NewUIState()
	t.input = NewInputReader()

	return t
}

// IsTerminal returns whether the TUI is running in a terminal.
// If false, the TUI operates in passthrough mode (plain output).
func (t *TUI) IsTerminal() bool {
	return t.isInteractiveTerminal()
}

func (t *TUI) isInteractiveTerminal() bool {
	return t.enabled && t.terminal.IsTerminal()
}

// Start initializes the TUI and begins the render loop.
// In non-TTY mode, this is a no-op.
// Returns an error if already running or if terminal setup fails.
func (t *TUI) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return nil // Already running
	}

	// In non-TTY or disabled mode, just mark as running (no setup needed)
	if !t.isInteractiveTerminal() {
		t.running = true
		return nil
	}

	// Enter raw mode for direct input handling
	if err := t.terminal.EnterRawMode(); err != nil {
		return err
	}

	// Set up renderer with terminal size
	w, h := t.terminal.Size()
	t.renderer.SetSize(w, h)

	// Initialize terminal display
	t.renderer.Init()
	t.renderer.SetScrollRegion()

	// Create stop channels
	t.stopChan = make(chan struct{})
	t.doneChan = make(chan struct{})
	t.running = true

	// Start background goroutines
	go t.renderLoop()
	go t.inputLoop()
	go t.resizeLoop()

	return nil
}

// Stop shuts down the TUI and restores the terminal.
// Safe to call multiple times.
func (t *TUI) Stop() {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.running = false

	// Signal goroutines to stop
	if t.stopChan != nil {
		close(t.stopChan)
	}
	t.mu.Unlock()

	// Wait for render loop to finish
	if t.doneChan != nil {
		<-t.doneChan
	}

	// Clean up terminal
	if t.isInteractiveTerminal() {
		t.renderer.Cleanup()
		t.terminal.StopWatchResize()
		t.terminal.Restore()
	}

	// Close input reader
	t.input.Close()
}

// renderLoop periodically redraws the UI
func (t *TUI) renderLoop() {
	defer close(t.doneChan)

	ticker := time.NewTicker(t.renderInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.render()
		}
	}
}

// render performs a single UI redraw
func (t *TUI) render() {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	// Get consistent state snapshot
	snapshot := t.state.Snapshot()
	snapshot.Paused = t.buffer.Paused() // Buffer owns pause state

	// Draw the UI
	t.renderer.Draw(t.buffer, snapshot)
}

// inputLoop reads keyboard input and handles controls
func (t *TUI) inputLoop() {
	for {
		// Check if stopped
		select {
		case <-t.stopChan:
			return
		default:
		}

		key, err := t.input.ReadKey()
		if err != nil {
			return
		}

		t.handleKey(key)
	}
}

// handleKey processes a keyboard input
func (t *TUI) handleKey(key Key) {
	_, h := t.terminal.Size()
	pageSize := t.renderer.LogAreaHeight()
	if pageSize <= 0 {
		pageSize = h - FooterHeight
	}

	switch key {
	case KeySpace:
		t.buffer.TogglePaused()
	case KeyUp, KeyK:
		t.buffer.SetPaused(true)
		t.buffer.ScrollUp(1)
	case KeyDown, KeyJ:
		t.buffer.ScrollDown(1)
	case KeyPageUp:
		t.buffer.SetPaused(true)
		t.buffer.ScrollUp(pageSize)
	case KeyPageDown:
		t.buffer.ScrollDown(pageSize)
	case KeyG:
		t.buffer.SetPaused(true)
		t.buffer.ScrollToTop()
	case KeyShiftG:
		t.buffer.SetPaused(false)
		t.buffer.ScrollToBottom()
	case KeyCtrlC:
		// Re-raise SIGINT so the application handles it properly
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(os.Interrupt)
	}
}

// resizeLoop watches for terminal resize events
func (t *TUI) resizeLoop() {
	resizeChan := t.terminal.WatchResize()

	for {
		select {
		case <-t.stopChan:
			return
		case <-resizeChan:
			w, h := t.terminal.Size()
			t.renderer.SetSize(w, h)
			t.renderer.SetScrollRegion()
			t.render() // Immediate redraw on resize
		}
	}
}

// WriteLine adds a line to the log buffer.
// In non-TTY mode, writes directly to stdout.
func (t *TUI) WriteLine(content string, lineType LineType) {
	if !t.isInteractiveTerminal() {
		// Passthrough mode: write directly
		t.renderer.WritePlain(Line{Content: content, LineType: lineType})
		return
	}

	t.buffer.Append(content, lineType)
}

// WriteLineDefault writes a line with default styling
func (t *TUI) WriteLineDefault(content string) {
	t.WriteLine(content, LineTypeDefault)
}

// WriteLineInfo writes an info-styled line
func (t *TUI) WriteLineInfo(content string) {
	t.WriteLine(content, LineTypeInfo)
}

// WriteLineSuccess writes a success-styled line
func (t *TUI) WriteLineSuccess(content string) {
	t.WriteLine(content, LineTypeSuccess)
}

// WriteLineWarning writes a warning-styled line
func (t *TUI) WriteLineWarning(content string) {
	t.WriteLine(content, LineTypeWarning)
}

// WriteLineError writes an error-styled line
func (t *TUI) WriteLineError(content string) {
	t.WriteLine(content, LineTypeError)
}

// SetPhase updates the current execution phase shown in the footer
func (t *TUI) SetPhase(phase Phase) {
	t.state.SetPhase(phase)
}

// SetIteration updates the current iteration number
func (t *TUI) SetIteration(iteration int) {
	t.state.SetIteration(iteration)
}

// SetTodos updates the todo progress shown in the footer
func (t *TUI) SetTodos(completed, total int) {
	t.state.SetTodos(completed, total)
}

// SetCurrentTodo updates the current in-progress todo text
func (t *TUI) SetCurrentTodo(todo string) {
	t.state.SetCurrentTodo(todo)
}

// SetTokens updates the token utilization shown in the footer
func (t *TUI) SetTokens(used, max int) {
	t.state.SetTokens(used, max)
}

// AddTokens adds to the cumulative token count
func (t *TUI) AddTokens(tokens int) {
	t.state.AddTokens(tokens)
}

// SetMaxTokens sets the maximum token limit
func (t *TUI) SetMaxTokens(max int) {
	used, _ := t.state.Tokens()
	t.state.SetTokens(used, max)
}

// Buffer returns the underlying buffer for advanced use cases
func (t *TUI) Buffer() *Buffer {
	return t.buffer
}

// State returns the underlying state for advanced use cases
func (t *TUI) State() *UIState {
	return t.state
}

// Run is a convenience method that starts the TUI and blocks until
// the context is cancelled, then stops the TUI.
func (t *TUI) Run(ctx context.Context) error {
	if err := t.Start(); err != nil {
		return err
	}

	<-ctx.Done()
	t.Stop()
	return ctx.Err()
}
