package testmode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Scenario defines the test behavior
type Scenario string

const (
	ScenarioSuccess Scenario = "success"
	ScenarioError   Scenario = "error"
	ScenarioPartial Scenario = "partial"
)

// MockClaude simulates Claude CLI behavior for test mode
type MockClaude struct {
	scenario  Scenario
	iteration int
	agentDir  string
	phase     string // "main", "code_review"
}

// NewMockClaude creates a new mock Claude client
func NewMockClaude(scenario string, agentDir string) *MockClaude {
	return &MockClaude{
		scenario: Scenario(scenario),
		agentDir: agentDir,
		phase:    "main",
	}
}

// SetPhase sets the current phase for appropriate todo generation
func (m *MockClaude) SetPhase(phase string) {
	m.phase = phase
	if phase == "code_review" {
		m.iteration = 0 // Reset iteration for code review phase
	}
}

// GetIteration returns the current iteration number
func (m *MockClaude) GetIteration() int {
	return m.iteration
}

// RunIteration simulates a single Claude iteration
// Returns (exitCode, error)
func (m *MockClaude) RunIteration(ctx context.Context) (int, error) {
	m.iteration++

	// Check context cancellation
	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	default:
	}

	// Simulate work duration (deterministic for testing)
	duration := time.Duration(m.iteration*100) * time.Millisecond
	time.Sleep(duration)

	// Handle error scenario
	if m.scenario == ScenarioError && m.iteration == 3 {
		return 1, fmt.Errorf("simulated Claude error on iteration 3")
	}

	// Update TODO.md based on phase and iteration
	if err := m.updateTodos(); err != nil {
		return -1, err
	}

	return 0, nil
}

// updateTodos writes appropriate TODO.md content for current iteration
func (m *MockClaude) updateTodos() error {
	todoPath := filepath.Join(m.agentDir, "TODO.md")

	var content string

	switch m.phase {
	case "main":
		content = m.getMainPhaseTodos()
	case "code_review":
		content = m.getCodeReviewTodos()
	default:
		content = m.getMainPhaseTodos()
	}

	return os.WriteFile(todoPath, []byte(content), 0644)
}

// getMainPhaseTodos returns TODO.md content for main loop iteration
func (m *MockClaude) getMainPhaseTodos() string {
	switch m.iteration {
	case 1:
		return `# Tasks

- [ ] Implement feature A
- [ ] Add tests for feature A
- [ ] Update documentation
`
	case 2:
		return `# Tasks

- [-] Implement feature A
- [ ] Add tests for feature A
- [ ] Update documentation
`
	case 3:
		return `# Tasks

- [x] Implement feature A
- [-] Add tests for feature A
- [ ] Update documentation
`
	case 4:
		if m.scenario == ScenarioPartial {
			// Partial scenario: max iterations hit before completion
			return `# Tasks

- [x] Implement feature A
- [x] Add tests for feature A
- [ ] Update documentation
`
		}
		return `# Tasks

- [x] Implement feature A
- [x] Add tests for feature A
- [-] Update documentation
`
	default: // 5+
		return `# Tasks

- [x] Implement feature A
- [x] Add tests for feature A
- [x] Update documentation
`
	}
}

// getCodeReviewTodos returns TODO.md content for code review phase
func (m *MockClaude) getCodeReviewTodos() string {
	switch m.iteration {
	case 1:
		return `# Code Review

## Issues Found

- [ ] Fix potential null pointer in handler.go:45
- [ ] Add error handling for database connection
`
	case 2:
		return `# Code Review

## Issues Found

- [x] Fix potential null pointer in handler.go:45
- [-] Add error handling for database connection
`
	default:
		return `# Code Review

## Issues Found

- [x] Fix potential null pointer in handler.go:45
- [x] Add error handling for database connection
`
	}
}

// StreamOutput simulates Claude's JSON stream output
// Returns a channel that yields simulated JSON lines
func (m *MockClaude) StreamOutput() <-chan string {
	lines := make(chan string, 10)
	go func() {
		defer close(lines)

		// Emit simulated output based on phase
		switch m.phase {
		case "main":
			lines <- fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"text","text":"[Test Mode] Working on iteration %d..."}]}}`, m.iteration)
		case "code_review":
			lines <- fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"text","text":"[Test Mode] Code review iteration %d..."}]}}`, m.iteration)
		}

		lines <- `{"type":"result","subtype":"success","result":"Test iteration complete"}`
	}()
	return lines
}

// AllTodosComplete returns true if all todos should be considered complete
func (m *MockClaude) AllTodosComplete() bool {
	if m.phase == "main" {
		if m.scenario == ScenarioPartial {
			return false // Partial never completes
		}
		return m.iteration >= 5
	}
	if m.phase == "code_review" {
		return m.iteration >= 3
	}
	return false
}
