package metrics

import (
	"context"
	"time"

	"github.com/hev/ralph/internal/todo"
)

// MetricsCollector defines the interface for collecting metrics.
// This interface enables mocking the metrics collector for testing.
type MetricsCollector interface {
	// RecordIterationComplete records the completion of an iteration
	RecordIterationComplete(ctx context.Context, duration time.Duration, exitReason string)

	// RecordCommits records the number of commits made
	RecordCommits(ctx context.Context, count int)

	// RecordError records an error occurrence
	RecordError(ctx context.Context, errType string)

	// UpdateTodoCounts updates the pending and completed todo counts
	UpdateTodoCounts(pending, completed int)

	// SessionStart marks the start of a session
	SessionStart(ctx context.Context)

	// SessionEnd marks the end of a session
	SessionEnd(ctx context.Context)

	// Shutdown performs cleanup and flushes any buffered metrics
	Shutdown(ctx context.Context) error

	// IsEnabled returns whether metrics collection is enabled
	IsEnabled() bool
}

// Ensure Collector implements MetricsCollector
var _ MetricsCollector = (*Collector)(nil)

// MetricsTracker defines the interface for high-level metrics tracking.
// This interface enables mocking the tracker for testing.
type MetricsTracker interface {
	// Start marks the session as started
	Start(ctx context.Context)

	// Stop marks the session as stopped and shuts down metrics
	Stop(ctx context.Context) error

	// BeforeIteration should be called before each iteration
	BeforeIteration()

	// AfterIteration should be called after each iteration completes
	AfterIteration(ctx context.Context, duration time.Duration, hadError bool, exitReason string)

	// RecordError records a Claude error
	RecordError(ctx context.Context, errorType string)

	// GetTodoCounts returns the current todo counts
	GetTodoCounts() (*todo.Counts, error)

	// GetCommitsDelta returns the number of commits since session start
	GetCommitsDelta() int

	// IsEnabled returns whether metrics tracking is enabled
	IsEnabled() bool

	// GetTodoItems returns the current todo items
	GetTodoItems() []todo.Item

	// GetNewlyCompletedTodos compares current todos with previous state and returns newly completed items
	GetNewlyCompletedTodos() []todo.Item

	// GetNewlyInProgressTodos compares current todos with previous state and returns newly in-progress items
	GetNewlyInProgressTodos() []TodoWithIndex

	// UpdatePreviousTodos saves the current todo state for later comparison
	UpdatePreviousTodos()
}

// Ensure Tracker implements MetricsTracker
var _ MetricsTracker = (*Tracker)(nil)
