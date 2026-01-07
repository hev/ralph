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
	collector    *Collector
	gitTracker   *git.Tracker
	agentDir     string
	iterationNum int
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
