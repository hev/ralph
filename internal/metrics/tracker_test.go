package metrics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/todo"
)

func TestTracker_GetTodoCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected *todo.Counts
	}{
		{
			name:    "empty file",
			content: "",
			expected: &todo.Counts{
				Pending:    0,
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "mixed statuses",
			content: "- [ ] Pending 1\n- [ ] Pending 2\n- [x] Done\n- [-] Working\n",
			expected: &todo.Counts{
				Pending:    2,
				InProgress: 1,
				Completed:  1,
			},
		},
		{
			name:    "all completed",
			content: "- [x] Task 1\n- [X] Task 2\n- [x] Task 3\n",
			expected: &todo.Counts{
				Pending:    0,
				InProgress: 0,
				Completed:  3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory with TODO.md
			tmpDir := t.TempDir()
			todoPath := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(todoPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write TODO.md: %v", err)
			}

			// Create tracker with disabled collector
			tracker := &Tracker{
				collector: &Collector{enabled: false},
				agentDir:  tmpDir,
			}

			// Get todo counts
			counts, err := tracker.GetTodoCounts()
			if err != nil {
				t.Fatalf("GetTodoCounts() error = %v", err)
			}

			// Verify counts
			if counts.Pending != tt.expected.Pending {
				t.Errorf("Pending = %d, want %d", counts.Pending, tt.expected.Pending)
			}
			if counts.InProgress != tt.expected.InProgress {
				t.Errorf("InProgress = %d, want %d", counts.InProgress, tt.expected.InProgress)
			}
			if counts.Completed != tt.expected.Completed {
				t.Errorf("Completed = %d, want %d", counts.Completed, tt.expected.Completed)
			}
		})
	}
}

func TestTracker_GetTodoCounts_NoFile(t *testing.T) {
	t.Parallel()

	// Create temp directory without TODO.md
	tmpDir := t.TempDir()

	tracker := &Tracker{
		collector: &Collector{enabled: false},
		agentDir:  tmpDir,
	}

	// Get todo counts - should return zeros, not error
	counts, err := tracker.GetTodoCounts()
	if err != nil {
		t.Fatalf("GetTodoCounts() error = %v, want nil for missing file", err)
	}

	if counts.Pending != 0 || counts.InProgress != 0 || counts.Completed != 0 {
		t.Errorf("Expected zero counts for missing file, got Pending=%d, InProgress=%d, Completed=%d",
			counts.Pending, counts.InProgress, counts.Completed)
	}
}

func TestTracker_GetTodoItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected []todo.Item
	}{
		{
			name:     "empty file",
			content:  "",
			expected: nil,
		},
		{
			name:    "mixed statuses",
			content: "- [ ] Pending task\n- [x] Completed task\n- [-] In progress task\n",
			expected: []todo.Item{
				{Text: "Pending task", Status: todo.StatusPending},
				{Text: "Completed task", Status: todo.StatusCompleted},
				{Text: "In progress task", Status: todo.StatusInProgress},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory with TODO.md
			tmpDir := t.TempDir()
			todoPath := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(todoPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write TODO.md: %v", err)
			}

			tracker := &Tracker{
				collector: &Collector{enabled: false},
				agentDir:  tmpDir,
			}

			items := tracker.GetTodoItems()

			if len(items) != len(tt.expected) {
				t.Fatalf("Got %d items, want %d", len(items), len(tt.expected))
			}

			for i, item := range items {
				if item.Text != tt.expected[i].Text {
					t.Errorf("Item[%d].Text = %q, want %q", i, item.Text, tt.expected[i].Text)
				}
				if item.Status != tt.expected[i].Status {
					t.Errorf("Item[%d].Status = %v, want %v", i, item.Status, tt.expected[i].Status)
				}
			}
		})
	}
}

func TestTracker_GetNewlyCompletedTodos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		previousTodos []todo.Item
		currentContent string
		expected      []todo.Item
	}{
		{
			name:          "no previous todos returns nil",
			previousTodos: nil,
			currentContent: "- [x] Done task\n",
			expected:      nil,
		},
		{
			name: "no changes",
			previousTodos: []todo.Item{
				{Text: "Task 1", Status: todo.StatusCompleted},
			},
			currentContent: "- [x] Task 1\n",
			expected:       nil,
		},
		{
			name: "one newly completed",
			previousTodos: []todo.Item{
				{Text: "Task 1", Status: todo.StatusPending},
				{Text: "Task 2", Status: todo.StatusPending},
			},
			currentContent: "- [x] Task 1\n- [ ] Task 2\n",
			expected: []todo.Item{
				{Text: "Task 1", Status: todo.StatusCompleted},
			},
		},
		{
			name: "multiple newly completed",
			previousTodos: []todo.Item{
				{Text: "Task 1", Status: todo.StatusPending},
				{Text: "Task 2", Status: todo.StatusInProgress},
				{Text: "Task 3", Status: todo.StatusPending},
			},
			currentContent: "- [x] Task 1\n- [x] Task 2\n- [x] Task 3\n",
			expected: []todo.Item{
				{Text: "Task 1", Status: todo.StatusCompleted},
				{Text: "Task 2", Status: todo.StatusCompleted},
				{Text: "Task 3", Status: todo.StatusCompleted},
			},
		},
		{
			name: "already completed not returned",
			previousTodos: []todo.Item{
				{Text: "Already done", Status: todo.StatusCompleted},
				{Text: "New done", Status: todo.StatusPending},
			},
			currentContent: "- [x] Already done\n- [x] New done\n",
			expected: []todo.Item{
				{Text: "New done", Status: todo.StatusCompleted},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory with TODO.md
			tmpDir := t.TempDir()
			todoPath := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(todoPath, []byte(tt.currentContent), 0644); err != nil {
				t.Fatalf("Failed to write TODO.md: %v", err)
			}

			tracker := &Tracker{
				collector:     &Collector{enabled: false},
				agentDir:      tmpDir,
				previousTodos: tt.previousTodos,
			}

			completed := tracker.GetNewlyCompletedTodos()

			if len(completed) != len(tt.expected) {
				t.Fatalf("Got %d items, want %d", len(completed), len(tt.expected))
			}

			for i, item := range completed {
				if item.Text != tt.expected[i].Text {
					t.Errorf("Item[%d].Text = %q, want %q", i, item.Text, tt.expected[i].Text)
				}
				if item.Status != tt.expected[i].Status {
					t.Errorf("Item[%d].Status = %v, want %v", i, item.Status, tt.expected[i].Status)
				}
			}
		})
	}
}

func TestTracker_GetNewlyInProgressTodos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		previousTodos []todo.Item
		currentContent string
		expected      []TodoWithIndex
	}{
		{
			name:          "no previous todos returns nil",
			previousTodos: nil,
			currentContent: "- [-] Working task\n",
			expected:      nil,
		},
		{
			name: "no changes",
			previousTodos: []todo.Item{
				{Text: "Task 1", Status: todo.StatusInProgress},
			},
			currentContent: "- [-] Task 1\n",
			expected:       nil,
		},
		{
			name: "one newly in progress",
			previousTodos: []todo.Item{
				{Text: "Task 1", Status: todo.StatusPending},
				{Text: "Task 2", Status: todo.StatusPending},
			},
			currentContent: "- [-] Task 1\n- [ ] Task 2\n",
			expected: []TodoWithIndex{
				{Item: todo.Item{Text: "Task 1", Status: todo.StatusInProgress}, Index: 1},
			},
		},
		{
			name: "multiple newly in progress with correct indices",
			previousTodos: []todo.Item{
				{Text: "Task 1", Status: todo.StatusPending},
				{Text: "Task 2", Status: todo.StatusPending},
				{Text: "Task 3", Status: todo.StatusPending},
			},
			currentContent: "- [-] Task 1\n- [ ] Task 2\n- [-] Task 3\n",
			expected: []TodoWithIndex{
				{Item: todo.Item{Text: "Task 1", Status: todo.StatusInProgress}, Index: 1},
				{Item: todo.Item{Text: "Task 3", Status: todo.StatusInProgress}, Index: 3},
			},
		},
		{
			name: "already in progress not returned",
			previousTodos: []todo.Item{
				{Text: "Already working", Status: todo.StatusInProgress},
				{Text: "New working", Status: todo.StatusPending},
			},
			currentContent: "- [-] Already working\n- [-] New working\n",
			expected: []TodoWithIndex{
				{Item: todo.Item{Text: "New working", Status: todo.StatusInProgress}, Index: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory with TODO.md
			tmpDir := t.TempDir()
			todoPath := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(todoPath, []byte(tt.currentContent), 0644); err != nil {
				t.Fatalf("Failed to write TODO.md: %v", err)
			}

			tracker := &Tracker{
				collector:     &Collector{enabled: false},
				agentDir:      tmpDir,
				previousTodos: tt.previousTodos,
			}

			inProgress := tracker.GetNewlyInProgressTodos()

			if len(inProgress) != len(tt.expected) {
				t.Fatalf("Got %d items, want %d", len(inProgress), len(tt.expected))
			}

			for i, item := range inProgress {
				if item.Item.Text != tt.expected[i].Item.Text {
					t.Errorf("Item[%d].Item.Text = %q, want %q", i, item.Item.Text, tt.expected[i].Item.Text)
				}
				if item.Item.Status != tt.expected[i].Item.Status {
					t.Errorf("Item[%d].Item.Status = %v, want %v", i, item.Item.Status, tt.expected[i].Item.Status)
				}
				if item.Index != tt.expected[i].Index {
					t.Errorf("Item[%d].Index = %d, want %d", i, item.Index, tt.expected[i].Index)
				}
			}
		})
	}
}

func TestTracker_UpdatePreviousTodos(t *testing.T) {
	t.Parallel()

	// Create temp directory with TODO.md
	tmpDir := t.TempDir()
	todoPath := filepath.Join(tmpDir, "TODO.md")
	content := "- [ ] Task 1\n- [x] Task 2\n- [-] Task 3\n"
	if err := os.WriteFile(todoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write TODO.md: %v", err)
	}

	tracker := &Tracker{
		collector: &Collector{enabled: false},
		agentDir:  tmpDir,
	}

	// Initially no previous todos
	if len(tracker.previousTodos) != 0 {
		t.Errorf("Expected empty previousTodos initially, got %d", len(tracker.previousTodos))
	}

	// Update previous todos
	tracker.UpdatePreviousTodos()

	// Verify snapshot was saved
	if len(tracker.previousTodos) != 3 {
		t.Fatalf("Expected 3 previousTodos after update, got %d", len(tracker.previousTodos))
	}

	expected := []todo.Item{
		{Text: "Task 1", Status: todo.StatusPending},
		{Text: "Task 2", Status: todo.StatusCompleted},
		{Text: "Task 3", Status: todo.StatusInProgress},
	}

	for i, item := range tracker.previousTodos {
		if item.Text != expected[i].Text {
			t.Errorf("previousTodos[%d].Text = %q, want %q", i, item.Text, expected[i].Text)
		}
		if item.Status != expected[i].Status {
			t.Errorf("previousTodos[%d].Status = %v, want %v", i, item.Status, expected[i].Status)
		}
	}
}

func TestTracker_GetCommitsDelta_NoGitTracker(t *testing.T) {
	t.Parallel()

	tracker := &Tracker{
		collector:  &Collector{enabled: false},
		gitTracker: nil,
	}

	delta := tracker.GetCommitsDelta()
	if delta != 0 {
		t.Errorf("GetCommitsDelta() with nil gitTracker = %d, want 0", delta)
	}
}

func TestTracker_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enabled collector",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disabled collector",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := &Tracker{
				collector: &Collector{enabled: tt.enabled},
			}

			if got := tracker.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTracker_BeforeIteration(t *testing.T) {
	t.Parallel()

	tracker := &Tracker{
		collector:    &Collector{enabled: false},
		iterationNum: 0,
		gitTracker:   nil,
	}

	// First iteration
	tracker.BeforeIteration()
	if tracker.iterationNum != 1 {
		t.Errorf("After first BeforeIteration(), iterationNum = %d, want 1", tracker.iterationNum)
	}

	// Second iteration
	tracker.BeforeIteration()
	if tracker.iterationNum != 2 {
		t.Errorf("After second BeforeIteration(), iterationNum = %d, want 2", tracker.iterationNum)
	}
}

func TestTracker_StartStop(t *testing.T) {
	t.Parallel()

	tracker := &Tracker{
		collector: &Collector{enabled: false},
	}

	ctx := context.Background()

	// Start should not panic with disabled collector
	tracker.Start(ctx)

	// Stop should not panic and should not error with disabled collector
	err := tracker.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}
}

func TestTracker_AfterIteration(t *testing.T) {
	t.Parallel()

	// Create temp directory with TODO.md
	tmpDir := t.TempDir()
	todoPath := filepath.Join(tmpDir, "TODO.md")
	content := "- [ ] Task 1\n- [x] Task 2\n"
	if err := os.WriteFile(todoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write TODO.md: %v", err)
	}

	tracker := &Tracker{
		collector:  &Collector{enabled: false},
		agentDir:   tmpDir,
		gitTracker: nil,
	}

	ctx := context.Background()
	duration := 5 * time.Second

	// Should not panic with disabled collector
	tracker.AfterIteration(ctx, duration, false, "completed")
	tracker.AfterIteration(ctx, duration, true, "error")
}

func TestTracker_RecordError(t *testing.T) {
	t.Parallel()

	tracker := &Tracker{
		collector: &Collector{enabled: false},
	}

	ctx := context.Background()

	// Should not panic with disabled collector
	tracker.RecordError(ctx, "test_error")
}

func TestNewTracker_Disabled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := &config.Config{
		OTELEnabled:   false,
		OTELEndpoint:  "localhost:4317",
		MetricsPrefix: "test",
		ProjectName:   "testproject",
		SessionID:     "test-session-123",
		AgentDir:      tmpDir,
	}

	tracker, err := NewTracker(cfg)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	if tracker == nil {
		t.Fatal("NewTracker() returned nil")
	}

	if tracker.IsEnabled() {
		t.Error("Expected tracker to be disabled")
	}

	if tracker.agentDir != tmpDir {
		t.Errorf("agentDir = %s, want %s", tracker.agentDir, tmpDir)
	}
}

func TestNewTracker_WithAgentDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	todoPath := filepath.Join(tmpDir, "TODO.md")
	content := "- [ ] Test task\n- [x] Done task\n"
	if err := os.WriteFile(todoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write TODO.md: %v", err)
	}

	cfg := &config.Config{
		OTELEnabled:   false,
		OTELEndpoint:  "localhost:4317",
		MetricsPrefix: "test",
		ProjectName:   "testproject",
		SessionID:     "test-session-123",
		AgentDir:      tmpDir,
	}

	tracker, err := NewTracker(cfg)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// Test that GetTodoCounts works with the created tracker
	counts, err := tracker.GetTodoCounts()
	if err != nil {
		t.Fatalf("GetTodoCounts() error = %v", err)
	}

	if counts.Pending != 1 {
		t.Errorf("Pending = %d, want 1", counts.Pending)
	}
	if counts.Completed != 1 {
		t.Errorf("Completed = %d, want 1", counts.Completed)
	}
}

func TestTracker_FullWorkflow(t *testing.T) {
	t.Parallel()

	// Create temp directory with TODO.md
	tmpDir := t.TempDir()
	todoPath := filepath.Join(tmpDir, "TODO.md")

	cfg := &config.Config{
		OTELEnabled:   false,
		OTELEndpoint:  "localhost:4317",
		MetricsPrefix: "test",
		ProjectName:   "testproject",
		SessionID:     "test-session-123",
		AgentDir:      tmpDir,
	}

	tracker, err := NewTracker(cfg)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	ctx := context.Background()

	// Start session
	tracker.Start(ctx)

	// Write initial TODO.md
	content := "- [ ] Task 1\n- [ ] Task 2\n- [ ] Task 3\n"
	if err := os.WriteFile(todoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write TODO.md: %v", err)
	}

	// Update previous todos
	tracker.UpdatePreviousTodos()

	// Simulate first iteration
	tracker.BeforeIteration()
	if tracker.iterationNum != 1 {
		t.Errorf("iterationNum = %d, want 1", tracker.iterationNum)
	}

	// Complete a task
	content = "- [x] Task 1\n- [-] Task 2\n- [ ] Task 3\n"
	if err := os.WriteFile(todoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write TODO.md: %v", err)
	}

	// Check newly completed
	newlyCompleted := tracker.GetNewlyCompletedTodos()
	if len(newlyCompleted) != 1 {
		t.Errorf("GetNewlyCompletedTodos() = %d items, want 1", len(newlyCompleted))
	}
	if len(newlyCompleted) > 0 && newlyCompleted[0].Text != "Task 1" {
		t.Errorf("Completed task = %q, want Task 1", newlyCompleted[0].Text)
	}

	// Check newly in progress
	newlyInProgress := tracker.GetNewlyInProgressTodos()
	if len(newlyInProgress) != 1 {
		t.Errorf("GetNewlyInProgressTodos() = %d items, want 1", len(newlyInProgress))
	}
	if len(newlyInProgress) > 0 {
		if newlyInProgress[0].Item.Text != "Task 2" {
			t.Errorf("In progress task = %q, want Task 2", newlyInProgress[0].Item.Text)
		}
		if newlyInProgress[0].Index != 2 {
			t.Errorf("In progress index = %d, want 2", newlyInProgress[0].Index)
		}
	}

	// After iteration
	tracker.AfterIteration(ctx, 5*time.Second, false, "completed")

	// Update previous todos for next iteration
	tracker.UpdatePreviousTodos()

	// Second iteration - now the same tasks should not appear as "newly" changed
	tracker.BeforeIteration()
	if tracker.iterationNum != 2 {
		t.Errorf("iterationNum = %d, want 2", tracker.iterationNum)
	}

	// No changes - should return empty
	newlyCompleted = tracker.GetNewlyCompletedTodos()
	if len(newlyCompleted) != 0 {
		t.Errorf("GetNewlyCompletedTodos() after no changes = %d items, want 0", len(newlyCompleted))
	}

	newlyInProgress = tracker.GetNewlyInProgressTodos()
	if len(newlyInProgress) != 0 {
		t.Errorf("GetNewlyInProgressTodos() after no changes = %d items, want 0", len(newlyInProgress))
	}

	// Stop session
	err = tracker.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestTracker_AfterIteration_WithError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	todoPath := filepath.Join(tmpDir, "TODO.md")
	content := "- [ ] Task 1\n"
	if err := os.WriteFile(todoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write TODO.md: %v", err)
	}

	tracker := &Tracker{
		collector:  &Collector{enabled: false},
		agentDir:   tmpDir,
		gitTracker: nil,
	}

	ctx := context.Background()

	// Should record error without panicking
	tracker.AfterIteration(ctx, 5*time.Second, true, "error")
}

func TestTracker_updateTodoMetrics_FileError(t *testing.T) {
	t.Parallel()

	// Use non-existent directory - should gracefully handle error
	tracker := &Tracker{
		collector: &Collector{enabled: false},
		agentDir:  "/nonexistent/path",
	}

	// Should not panic - silently returns on error
	tracker.updateTodoMetrics()
}

func TestTracker_GetTodoItems_NoFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// No TODO.md file

	tracker := &Tracker{
		collector: &Collector{enabled: false},
		agentDir:  tmpDir,
	}

	items := tracker.GetTodoItems()
	if items != nil {
		t.Errorf("GetTodoItems() with no file = %v, want nil", items)
	}
}
