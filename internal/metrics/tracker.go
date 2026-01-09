package metrics

import (
	"context"
	"path/filepath"
	"time"

	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/git"
	"github.com/hev/ralph/internal/todo"
)

// Tracker provides high-level metrics tracking helpers
type Tracker struct {
	collector     *Collector
	gitTracker    *git.Tracker
	agentDir      string
	iterationNum  int
	previousTodos []todo.Item // Previous iteration's todo items
}

// NewTracker creates a new metrics tracker
func NewTracker(cfg *config.Config) (*Tracker, error) {
	collector, err := NewCollector(CollectorConfig{
		Enabled:       cfg.OTELEnabled,
		Endpoint:      cfg.OTELEndpoint,
		MetricsPrefix: cfg.MetricsPrefix,
		ProjectName:   cfg.ProjectName,
		SessionID:     cfg.SessionID,
	})
	if err != nil {
		return nil, err
	}

	gitTracker, _ := git.NewTracker() // Ignore error - git tracking is optional

	return &Tracker{
		collector:  collector,
		gitTracker: gitTracker,
		agentDir:   cfg.AgentDir,
	}, nil
}

// Start marks the session as started
func (t *Tracker) Start(ctx context.Context) {
	t.collector.SessionStart(ctx)
}

// Stop marks the session as stopped and shuts down metrics
func (t *Tracker) Stop(ctx context.Context) error {
	t.collector.SessionEnd(ctx)
	return t.collector.Shutdown(ctx)
}

// BeforeIteration should be called before each iteration
func (t *Tracker) BeforeIteration() {
	t.iterationNum++
	// Update git baseline for per-iteration commit tracking
	if t.gitTracker != nil {
		t.gitTracker.UpdateBaseline()
	}
}

// AfterIteration should be called after each iteration completes
func (t *Tracker) AfterIteration(ctx context.Context, duration time.Duration, hadError bool, exitReason string) {
	// Record iteration metrics
	if hadError {
		t.collector.RecordIterationComplete(ctx, duration, "error")
		t.collector.RecordError(ctx, "execution_error")
	} else {
		t.collector.RecordIterationComplete(ctx, duration, exitReason)
	}

	// Track git commits
	if t.gitTracker != nil {
		delta, err := t.gitTracker.CommitsDelta()
		if err == nil && delta > 0 {
			t.collector.RecordCommits(ctx, delta)
		}
	}

	// Update todo counts
	t.updateTodoMetrics()
}

// RecordError records a Claude error
func (t *Tracker) RecordError(ctx context.Context, errorType string) {
	t.collector.RecordError(ctx, errorType)
}

// updateTodoMetrics parses TODO.md and updates metrics
func (t *Tracker) updateTodoMetrics() {
	todoPath := filepath.Join(t.agentDir, "TODO.md")
	counts, err := todo.ParseFile(todoPath)
	if err != nil {
		return
	}

	t.collector.UpdateTodoCounts(counts.Pending, counts.Completed)
}

// GetTodoCounts returns the current todo counts
func (t *Tracker) GetTodoCounts() (*todo.Counts, error) {
	todoPath := filepath.Join(t.agentDir, "TODO.md")
	return todo.ParseFile(todoPath)
}

// GetCommitsDelta returns the number of commits since session start
func (t *Tracker) GetCommitsDelta() int {
	if t.gitTracker == nil {
		return 0
	}
	delta, _ := t.gitTracker.CommitsDelta()
	return delta
}

// IsEnabled returns whether metrics tracking is enabled
func (t *Tracker) IsEnabled() bool {
	return t.collector.IsEnabled()
}

// GetTodoItems returns the current todo items
func (t *Tracker) GetTodoItems() []todo.Item {
	todoPath := filepath.Join(t.agentDir, "TODO.md")
	items, _ := todo.ParseItems(todoPath)
	return items
}

// GetNewlyCompletedTodos compares current todos with previous state and returns newly completed items
func (t *Tracker) GetNewlyCompletedTodos() []todo.Item {
	currentItems := t.GetTodoItems()
	if len(t.previousTodos) == 0 {
		return nil
	}

	// Build a set of previously completed todo texts
	previouslyCompleted := make(map[string]bool)
	for _, item := range t.previousTodos {
		if item.IsCompleted() {
			previouslyCompleted[item.Text] = true
		}
	}

	// Find items that are now completed but weren't before
	var newlyCompleted []todo.Item
	for _, item := range currentItems {
		if item.IsCompleted() && !previouslyCompleted[item.Text] {
			newlyCompleted = append(newlyCompleted, item)
		}
	}

	return newlyCompleted
}

// GetNewlyInProgressTodos compares current todos with previous state and returns newly in-progress items
func (t *Tracker) GetNewlyInProgressTodos() []todo.Item {
	currentItems := t.GetTodoItems()
	if len(t.previousTodos) == 0 {
		return nil
	}

	// Build a set of previously in-progress todo texts
	previouslyInProgress := make(map[string]bool)
	for _, item := range t.previousTodos {
		if item.IsInProgress() {
			previouslyInProgress[item.Text] = true
		}
	}

	// Find items that are now in-progress but weren't before
	var newlyInProgress []todo.Item
	for _, item := range currentItems {
		if item.IsInProgress() && !previouslyInProgress[item.Text] {
			newlyInProgress = append(newlyInProgress, item)
		}
	}

	return newlyInProgress
}

// UpdatePreviousTodos saves the current todo state for later comparison
func (t *Tracker) UpdatePreviousTodos() {
	t.previousTodos = t.GetTodoItems()
}
