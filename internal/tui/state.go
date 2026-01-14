package tui

import "sync"

// Phase represents the current execution phase of the runner
type Phase int

const (
	PhaseIdle Phase = iota
	PhaseMainLoop
	PhaseCodeReview
	PhaseCleanup
	PhasePR
)

// String returns the display name for a phase
func (p Phase) String() string {
	switch p {
	case PhaseMainLoop:
		return "Main Loop"
	case PhaseCodeReview:
		return "Code Review"
	case PhaseCleanup:
		return "Cleanup"
	case PhasePR:
		return "PR Creation"
	default:
		return "Idle"
	}
}

// UIState holds the current state of the TUI, updated by the runner
// and read by the renderer. All access is thread-safe.
type UIState struct {
	mu sync.RWMutex

	// Execution state
	phase     Phase
	iteration int

	// Todo progress
	todosCompleted int
	todosTotal     int
	currentTodo    string

	// Token utilization
	tokensUsed int
	tokensMax  int

	// UI state
	paused bool

	// Model information
	model string
}

// NewUIState creates a new UIState with default values
func NewUIState() *UIState {
	return &UIState{
		phase:     PhaseIdle,
		tokensMax: 200000, // Default context window
	}
}

// Phase returns the current execution phase
func (s *UIState) Phase() Phase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phase
}

// SetPhase updates the current execution phase
func (s *UIState) SetPhase(phase Phase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phase = phase
}

// Iteration returns the current iteration number
func (s *UIState) Iteration() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.iteration
}

// SetIteration updates the current iteration number
func (s *UIState) SetIteration(iteration int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.iteration = iteration
}

// Todos returns the todo progress (completed, total)
func (s *UIState) Todos() (completed, total int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.todosCompleted, s.todosTotal
}

// SetTodos updates the todo progress
func (s *UIState) SetTodos(completed, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todosCompleted = completed
	s.todosTotal = total
}

// CurrentTodo returns the current in-progress todo text
func (s *UIState) CurrentTodo() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTodo
}

// SetCurrentTodo updates the current in-progress todo text
func (s *UIState) SetCurrentTodo(todo string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTodo = todo
}

// Tokens returns the token utilization (used, max)
func (s *UIState) Tokens() (used, max int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokensUsed, s.tokensMax
}

// SetTokens updates the token utilization
func (s *UIState) SetTokens(used, max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokensUsed = used
	s.tokensMax = max
}

// AddTokens adds to the cumulative token count
func (s *UIState) AddTokens(tokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokensUsed += tokens
}

// Paused returns whether the UI is paused (auto-scroll disabled)
func (s *UIState) Paused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paused
}

// SetPaused sets the paused state
func (s *UIState) SetPaused(paused bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = paused
}

// TogglePaused toggles the paused state and returns the new state
func (s *UIState) TogglePaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = !s.paused
	return s.paused
}

// Model returns the current model name
func (s *UIState) Model() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.model
}

// SetModel updates the current model name
func (s *UIState) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
}

// Snapshot returns a point-in-time copy of all state values
// for consistent rendering without holding the lock
type StateSnapshot struct {
	Phase          Phase
	Iteration      int
	TodosCompleted int
	TodosTotal     int
	CurrentTodo    string
	TokensUsed     int
	TokensMax      int
	Paused         bool
	Model          string
}

// Snapshot returns a consistent snapshot of all state
func (s *UIState) Snapshot() StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return StateSnapshot{
		Phase:          s.phase,
		Iteration:      s.iteration,
		TodosCompleted: s.todosCompleted,
		TodosTotal:     s.todosTotal,
		CurrentTodo:    s.currentTodo,
		TokensUsed:     s.tokensUsed,
		TokensMax:      s.tokensMax,
		Paused:         s.paused,
		Model:          s.model,
	}
}
