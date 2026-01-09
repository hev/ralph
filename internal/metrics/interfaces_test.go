package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/hev/ralph/internal/todo"
)

// MockCollector implements MetricsCollector for testing
type MockCollector struct {
	enabled            bool
	iterationCount     int
	commitCount        int
	errorCount         int
	pendingCount       int
	completedCount     int
	sessionStartCalled bool
	sessionEndCalled   bool
	shutdownCalled     bool
}

func (m *MockCollector) RecordIterationComplete(ctx context.Context, duration time.Duration, exitReason string) {
	m.iterationCount++
}

func (m *MockCollector) RecordCommits(ctx context.Context, count int) {
	m.commitCount += count
}

func (m *MockCollector) RecordError(ctx context.Context, errType string) {
	m.errorCount++
}

func (m *MockCollector) UpdateTodoCounts(pending, completed int) {
	m.pendingCount = pending
	m.completedCount = completed
}

func (m *MockCollector) SessionStart(ctx context.Context) {
	m.sessionStartCalled = true
}

func (m *MockCollector) SessionEnd(ctx context.Context) {
	m.sessionEndCalled = true
}

func (m *MockCollector) Shutdown(ctx context.Context) error {
	m.shutdownCalled = true
	return nil
}

func (m *MockCollector) IsEnabled() bool {
	return m.enabled
}

func TestMockCollector_ImplementsMetricsCollector(t *testing.T) {
	t.Parallel()

	// Verify MockCollector implements MetricsCollector
	var _ MetricsCollector = (*MockCollector)(nil)
}

func TestMockCollector_RecordIterationComplete(t *testing.T) {
	t.Parallel()

	mock := &MockCollector{}
	ctx := context.Background()

	mock.RecordIterationComplete(ctx, 5*time.Second, "completed")
	mock.RecordIterationComplete(ctx, 10*time.Second, "error")

	if mock.iterationCount != 2 {
		t.Errorf("iterationCount = %d, want 2", mock.iterationCount)
	}
}

func TestMockCollector_RecordCommits(t *testing.T) {
	t.Parallel()

	mock := &MockCollector{}
	ctx := context.Background()

	mock.RecordCommits(ctx, 3)
	mock.RecordCommits(ctx, 5)

	if mock.commitCount != 8 {
		t.Errorf("commitCount = %d, want 8", mock.commitCount)
	}
}

func TestMockCollector_RecordError(t *testing.T) {
	t.Parallel()

	mock := &MockCollector{}
	ctx := context.Background()

	mock.RecordError(ctx, "test_error")
	mock.RecordError(ctx, "another_error")

	if mock.errorCount != 2 {
		t.Errorf("errorCount = %d, want 2", mock.errorCount)
	}
}

func TestMockCollector_UpdateTodoCounts(t *testing.T) {
	t.Parallel()

	mock := &MockCollector{}

	mock.UpdateTodoCounts(10, 5)

	if mock.pendingCount != 10 {
		t.Errorf("pendingCount = %d, want 10", mock.pendingCount)
	}
	if mock.completedCount != 5 {
		t.Errorf("completedCount = %d, want 5", mock.completedCount)
	}
}

func TestMockCollector_SessionLifecycle(t *testing.T) {
	t.Parallel()

	mock := &MockCollector{}
	ctx := context.Background()

	if mock.sessionStartCalled {
		t.Error("sessionStartCalled should be false initially")
	}
	if mock.sessionEndCalled {
		t.Error("sessionEndCalled should be false initially")
	}
	if mock.shutdownCalled {
		t.Error("shutdownCalled should be false initially")
	}

	mock.SessionStart(ctx)
	if !mock.sessionStartCalled {
		t.Error("sessionStartCalled should be true after SessionStart")
	}

	mock.SessionEnd(ctx)
	if !mock.sessionEndCalled {
		t.Error("sessionEndCalled should be true after SessionEnd")
	}

	err := mock.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
	if !mock.shutdownCalled {
		t.Error("shutdownCalled should be true after Shutdown")
	}
}

func TestMockCollector_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockCollector{enabled: tt.enabled}
			if got := mock.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// MockTracker implements MetricsTracker for testing
type MockTracker struct {
	started              bool
	stopped              bool
	beforeIterationCount int
	afterIterationCount  int
	errorCount           int
	todoItems            []todo.Item
	previousTodos        []todo.Item
	enabled              bool
}

func (m *MockTracker) Start(ctx context.Context) {
	m.started = true
}

func (m *MockTracker) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

func (m *MockTracker) BeforeIteration() {
	m.beforeIterationCount++
}

func (m *MockTracker) AfterIteration(ctx context.Context, duration time.Duration, hadError bool, exitReason string) {
	m.afterIterationCount++
}

func (m *MockTracker) RecordError(ctx context.Context, errorType string) {
	m.errorCount++
}

func (m *MockTracker) GetTodoCounts() (*todo.Counts, error) {
	pending := 0
	completed := 0
	inProgress := 0
	for _, item := range m.todoItems {
		switch item.Status {
		case todo.StatusPending:
			pending++
		case todo.StatusCompleted:
			completed++
		case todo.StatusInProgress:
			inProgress++
		}
	}
	return &todo.Counts{Pending: pending, Completed: completed, InProgress: inProgress}, nil
}

func (m *MockTracker) GetCommitsDelta() int {
	return 0
}

func (m *MockTracker) IsEnabled() bool {
	return m.enabled
}

func (m *MockTracker) GetTodoItems() []todo.Item {
	return m.todoItems
}

func (m *MockTracker) GetNewlyCompletedTodos() []todo.Item {
	return nil
}

func (m *MockTracker) GetNewlyInProgressTodos() []TodoWithIndex {
	return nil
}

func (m *MockTracker) UpdatePreviousTodos() {
	m.previousTodos = m.todoItems
}

func TestMockTracker_ImplementsMetricsTracker(t *testing.T) {
	t.Parallel()

	// Verify MockTracker implements MetricsTracker
	var _ MetricsTracker = (*MockTracker)(nil)
}

func TestMockTracker_Lifecycle(t *testing.T) {
	t.Parallel()

	mock := &MockTracker{}
	ctx := context.Background()

	if mock.started {
		t.Error("started should be false initially")
	}
	if mock.stopped {
		t.Error("stopped should be false initially")
	}

	mock.Start(ctx)
	if !mock.started {
		t.Error("started should be true after Start")
	}

	err := mock.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
	if !mock.stopped {
		t.Error("stopped should be true after Stop")
	}
}

func TestMockTracker_Iterations(t *testing.T) {
	t.Parallel()

	mock := &MockTracker{}
	ctx := context.Background()

	mock.BeforeIteration()
	mock.BeforeIteration()
	mock.AfterIteration(ctx, time.Second, false, "completed")

	if mock.beforeIterationCount != 2 {
		t.Errorf("beforeIterationCount = %d, want 2", mock.beforeIterationCount)
	}
	if mock.afterIterationCount != 1 {
		t.Errorf("afterIterationCount = %d, want 1", mock.afterIterationCount)
	}
}

func TestMockTracker_GetTodoCounts(t *testing.T) {
	t.Parallel()

	mock := &MockTracker{
		todoItems: []todo.Item{
			{Text: "Task 1", Status: todo.StatusPending},
			{Text: "Task 2", Status: todo.StatusCompleted},
			{Text: "Task 3", Status: todo.StatusInProgress},
			{Text: "Task 4", Status: todo.StatusPending},
		},
	}

	counts, err := mock.GetTodoCounts()
	if err != nil {
		t.Fatalf("GetTodoCounts() error = %v", err)
	}

	if counts.Pending != 2 {
		t.Errorf("Pending = %d, want 2", counts.Pending)
	}
	if counts.Completed != 1 {
		t.Errorf("Completed = %d, want 1", counts.Completed)
	}
	if counts.InProgress != 1 {
		t.Errorf("InProgress = %d, want 1", counts.InProgress)
	}
}

func TestTodoWithIndex(t *testing.T) {
	t.Parallel()

	item := todo.Item{Text: "Test task", Status: todo.StatusInProgress}
	todoWithIndex := TodoWithIndex{Item: item, Index: 5}

	if todoWithIndex.Item.Text != "Test task" {
		t.Errorf("Item.Text = %q, want 'Test task'", todoWithIndex.Item.Text)
	}
	if todoWithIndex.Item.Status != todo.StatusInProgress {
		t.Errorf("Item.Status = %v, want StatusInProgress", todoWithIndex.Item.Status)
	}
	if todoWithIndex.Index != 5 {
		t.Errorf("Index = %d, want 5", todoWithIndex.Index)
	}
}

func TestTrackerImplementsMetricsTracker(t *testing.T) {
	t.Parallel()

	// Verify Tracker implements MetricsTracker
	var _ MetricsTracker = (*Tracker)(nil)
}

func TestCollectorImplementsMetricsCollector(t *testing.T) {
	t.Parallel()

	// Verify Collector implements MetricsCollector
	var _ MetricsCollector = (*Collector)(nil)
}
