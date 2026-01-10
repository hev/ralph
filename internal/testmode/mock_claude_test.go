package testmode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewMockClaude(t *testing.T) {
	mock := NewMockClaude("success", "/tmp/test-agent")

	if mock.scenario != ScenarioSuccess {
		t.Errorf("expected scenario success, got %s", mock.scenario)
	}
	if mock.agentDir != "/tmp/test-agent" {
		t.Errorf("expected agentDir /tmp/test-agent, got %s", mock.agentDir)
	}
	if mock.phase != "main" {
		t.Errorf("expected phase main, got %s", mock.phase)
	}
	if mock.iteration != 0 {
		t.Errorf("expected iteration 0, got %d", mock.iteration)
	}
}

func TestMockClaude_SuccessScenario(t *testing.T) {
	tmpDir := t.TempDir()
	mock := NewMockClaude("success", tmpDir)

	ctx := context.Background()

	// Run through all iterations
	for i := 1; i <= 5; i++ {
		exitCode, err := mock.RunIteration(ctx)
		if err != nil {
			t.Errorf("Iteration %d failed: %v", i, err)
		}
		if exitCode != 0 {
			t.Errorf("Iteration %d exit code = %d, want 0", i, exitCode)
		}

		// Verify TODO.md was created
		todoPath := filepath.Join(tmpDir, "TODO.md")
		if _, err := os.Stat(todoPath); os.IsNotExist(err) {
			t.Errorf("TODO.md not created after iteration %d", i)
		}

		// Verify iteration counter
		if mock.GetIteration() != i {
			t.Errorf("expected iteration %d, got %d", i, mock.GetIteration())
		}
	}

	// Verify all todos complete
	if !mock.AllTodosComplete() {
		t.Error("expected AllTodosComplete to return true after 5 iterations")
	}
}

func TestMockClaude_ErrorScenario(t *testing.T) {
	tmpDir := t.TempDir()
	mock := NewMockClaude("error", tmpDir)

	ctx := context.Background()

	// Iteration 3 should return an error
	for i := 1; i <= 3; i++ {
		exitCode, err := mock.RunIteration(ctx)
		if i == 3 {
			if err == nil {
				t.Error("Iteration 3 should return error")
			}
			if exitCode != 1 {
				t.Errorf("Iteration 3 exit code = %d, want 1", exitCode)
			}
		} else {
			if err != nil {
				t.Errorf("Iteration %d should not error: %v", i, err)
			}
		}
	}
}

func TestMockClaude_PartialScenario(t *testing.T) {
	tmpDir := t.TempDir()
	mock := NewMockClaude("partial", tmpDir)

	ctx := context.Background()

	// Run through iterations
	for i := 1; i <= 5; i++ {
		exitCode, err := mock.RunIteration(ctx)
		if err != nil {
			t.Errorf("Iteration %d failed: %v", i, err)
		}
		if exitCode != 0 {
			t.Errorf("Iteration %d exit code = %d, want 0", i, exitCode)
		}
	}

	// Partial scenario should never complete all todos
	if mock.AllTodosComplete() {
		t.Error("partial scenario should not complete all todos")
	}
}

func TestMockClaude_CodeReviewPhase(t *testing.T) {
	tmpDir := t.TempDir()
	mock := NewMockClaude("success", tmpDir)
	mock.SetPhase("code_review")

	// Verify phase was set and iteration reset
	if mock.phase != "code_review" {
		t.Errorf("expected phase code_review, got %s", mock.phase)
	}
	if mock.GetIteration() != 0 {
		t.Errorf("expected iteration to reset to 0, got %d", mock.GetIteration())
	}

	ctx := context.Background()

	// Run code review iterations
	for i := 1; i <= 3; i++ {
		exitCode, err := mock.RunIteration(ctx)
		if err != nil {
			t.Errorf("Code review iteration %d failed: %v", i, err)
		}
		if exitCode != 0 {
			t.Errorf("Code review iteration %d exit code = %d, want 0", i, exitCode)
		}

		// Verify TODO.md was created
		todoPath := filepath.Join(tmpDir, "TODO.md")
		content, err := os.ReadFile(todoPath)
		if err != nil {
			t.Errorf("Failed to read TODO.md: %v", err)
		}

		// Verify content contains code review issues
		if !strings.Contains(string(content), "Code Review") {
			t.Errorf("TODO.md should contain 'Code Review' header")
		}
	}

	// Verify all code review todos complete
	if !mock.AllTodosComplete() {
		t.Error("expected AllTodosComplete to return true after 3 code review iterations")
	}
}

func TestMockClaude_TodoContent(t *testing.T) {
	tmpDir := t.TempDir()
	mock := NewMockClaude("success", tmpDir)
	ctx := context.Background()

	// Iteration 1: All pending
	mock.RunIteration(ctx)
	content := readTodoFile(t, tmpDir)
	if !strings.Contains(content, "- [ ] Implement feature A") {
		t.Error("Iteration 1: expected pending todo for 'Implement feature A'")
	}

	// Iteration 2: First in progress
	mock.RunIteration(ctx)
	content = readTodoFile(t, tmpDir)
	if !strings.Contains(content, "- [-] Implement feature A") {
		t.Error("Iteration 2: expected in-progress todo for 'Implement feature A'")
	}

	// Iteration 3: First complete, second in progress
	mock.RunIteration(ctx)
	content = readTodoFile(t, tmpDir)
	if !strings.Contains(content, "- [x] Implement feature A") {
		t.Error("Iteration 3: expected completed todo for 'Implement feature A'")
	}
	if !strings.Contains(content, "- [-] Add tests for feature A") {
		t.Error("Iteration 3: expected in-progress todo for 'Add tests for feature A'")
	}
}

func TestMockClaude_StreamOutput(t *testing.T) {
	mock := NewMockClaude("success", t.TempDir())

	// Run one iteration first
	mock.RunIteration(context.Background())

	// Get stream output
	lines := mock.StreamOutput()
	var collected []string
	for line := range lines {
		collected = append(collected, line)
	}

	if len(collected) != 2 {
		t.Errorf("expected 2 output lines, got %d", len(collected))
	}

	// Check first line contains iteration info
	if !strings.Contains(collected[0], "iteration") {
		t.Error("first line should contain 'iteration'")
	}

	// Check second line is result
	if !strings.Contains(collected[1], "result") {
		t.Error("second line should contain 'result'")
	}
}

func TestMockClaude_ContextCancellation(t *testing.T) {
	mock := NewMockClaude("success", t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	exitCode, err := mock.RunIteration(ctx)
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
	if exitCode != -1 {
		t.Errorf("expected exit code -1, got %d", exitCode)
	}
}

// Helper function to read TODO.md content
func readTodoFile(t *testing.T, agentDir string) string {
	t.Helper()
	todoPath := filepath.Join(agentDir, "TODO.md")
	content, err := os.ReadFile(todoPath)
	if err != nil {
		t.Fatalf("Failed to read TODO.md: %v", err)
	}
	return string(content)
}
