package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hev/ralph/internal/todo"
)

// Manager handles all state file operations for a Ralph session
type Manager struct {
	agentDir      string
	sessionID     string
	projectName   string
	startTime     time.Time
	runsRetention int
	enabled       bool
	provider      string
	mu            sync.Mutex

	// Internal tracking for run data
	runData *RunData
}

// Config holds configuration for the state manager
type Config struct {
	AgentDir      string
	SessionID     string
	ProjectName   string
	RunsRetention int
	Enabled       bool
	Provider      string
}

// NewManager creates a new state manager
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		agentDir:      cfg.AgentDir,
		sessionID:     cfg.SessionID,
		projectName:   cfg.ProjectName,
		startTime:     time.Now(),
		runsRetention: cfg.RunsRetention,
		enabled:       cfg.Enabled,
		provider:      cfg.Provider,
	}

	if m.runsRetention == 0 {
		m.runsRetention = 50
	}

	if !m.enabled {
		return m, nil
	}

	// Ensure directories exist
	if err := m.ensureDirectories(); err != nil {
		return nil, err
	}

	// Initialize run data
	m.runData = &RunData{
		SessionID:   m.sessionID,
		ProjectName: m.projectName,
		StartTime:   m.startTime,
		Iterations:  []IterationData{},
		Errors:      []ErrorData{},
	}

	return m, nil
}

// ensureDirectories creates the required directory structure
func (m *Manager) ensureDirectories() error {
	dirs := []string{
		m.agentDir,
		filepath.Join(m.agentDir, "runs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// LogSessionStart logs the start of a session to activity.log
func (m *Manager) LogSessionStart() {
	if !m.enabled {
		return
	}
	m.logActivity("SESSION_START", map[string]string{
		"session": m.sessionID,
		"project": m.projectName,
	})
}

// LogSessionEnd logs the end of a session and saves run data
func (m *Manager) LogSessionEnd(reason string, totalIterations int, totalCommits int) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	m.runData.EndTime = time.Now()
	m.runData.ExitReason = reason
	m.runData.TotalIterations = totalIterations
	m.runData.TotalCommits = totalCommits
	m.mu.Unlock()

	m.logActivity("SESSION_END", map[string]string{
		"reason":     reason,
		"duration":   time.Since(m.startTime).Round(time.Second).String(),
		"iterations": fmt.Sprintf("%d", totalIterations),
		"commits":    fmt.Sprintf("%d", totalCommits),
	})

	// Save run data
	m.saveRunData()

	// Cleanup old runs
	m.cleanupOldRuns()
}

// LogIterationStart logs the start of an iteration
func (m *Manager) LogIterationStart(iteration int) {
	if !m.enabled {
		return
	}
	m.logActivity("ITERATION_START", map[string]string{
		"iteration": fmt.Sprintf("%d", iteration),
	})

	m.mu.Lock()
	m.runData.Iterations = append(m.runData.Iterations, IterationData{
		Number:    iteration,
		StartTime: time.Now(),
	})
	m.mu.Unlock()
}

// LogIterationEnd logs the end of an iteration
func (m *Manager) LogIterationEnd(iteration int, duration time.Duration, exitCode int, commits int) {
	if !m.enabled {
		return
	}
	m.logActivity("ITERATION_END", map[string]string{
		"iteration": fmt.Sprintf("%d", iteration),
		"duration":  duration.Round(time.Second).String(),
		"exit_code": fmt.Sprintf("%d", exitCode),
		"commits":   fmt.Sprintf("%d", commits),
	})

	m.mu.Lock()
	if len(m.runData.Iterations) > 0 {
		idx := len(m.runData.Iterations) - 1
		m.runData.Iterations[idx].Duration = duration
		m.runData.Iterations[idx].ExitCode = exitCode
		m.runData.Iterations[idx].Commits = commits
	}
	m.mu.Unlock()
}

// LogProgress logs a completed task to progress.md
func (m *Manager) LogProgress(item todo.Item, duration time.Duration, commits int) {
	if !m.enabled {
		return
	}
	m.appendProgress(ProgressEntry{
		Task:     item.Text,
		Duration: duration,
		Commits:  commits,
		Time:     time.Now(),
	})

	m.mu.Lock()
	if len(m.runData.Iterations) > 0 {
		idx := len(m.runData.Iterations) - 1
		m.runData.Iterations[idx].TodosCompleted = append(
			m.runData.Iterations[idx].TodosCompleted,
			item.Text,
		)
	}
	m.mu.Unlock()
}

// LogError logs an error to errors.log
func (m *Manager) LogError(iteration int, exitCode int, phase string, context string) {
	if !m.enabled {
		return
	}
	m.appendError(ErrorEntry{
		Iteration: iteration,
		ExitCode:  exitCode,
		Phase:     phase,
		Context:   context,
		Time:      time.Now(),
	})

	m.mu.Lock()
	m.runData.Errors = append(m.runData.Errors, ErrorData{
		Iteration: iteration,
		ExitCode:  exitCode,
		Phase:     phase,
		Time:      time.Now(),
	})
	m.mu.Unlock()
}

// SetInitialTodos records the initial todo list for run data
func (m *Manager) SetInitialTodos(todos []todo.Item) {
	if !m.enabled {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runData.InitialTodos = make([]string, len(todos))
	for i, t := range todos {
		m.runData.InitialTodos[i] = t.Text
	}
}

// Paths returns the paths to state files
func (m *Manager) Paths() StatePaths {
	return StatePaths{
		AgentDir:           m.agentDir,
		ImplementationPlan: filepath.Join(m.agentDir, "IMPLEMENTATION_PLAN.md"),
		TODO:               filepath.Join(m.agentDir, "TODO.md"),
		Progress:           filepath.Join(m.agentDir, "progress.md"),
		Guardrails:         filepath.Join(m.agentDir, "guardrails.md"),
		ActivityLog:        filepath.Join(m.agentDir, "activity.log"),
		ErrorsLog:          filepath.Join(m.agentDir, "errors.log"),
		RunsDir:            filepath.Join(m.agentDir, "runs"),
	}
}

// StatePaths holds paths to all state files
type StatePaths struct {
	AgentDir           string
	ImplementationPlan string
	TODO               string
	Progress           string
	Guardrails         string
	ActivityLog        string
	ErrorsLog          string
	RunsDir            string
}
