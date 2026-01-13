package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunData holds all data for a single Ralph session run
type RunData struct {
	SessionID       string          `json:"session_id"`
	ProjectName     string          `json:"project_name"`
	StartTime       time.Time       `json:"start_time"`
	EndTime         time.Time       `json:"end_time"`
	ExitReason      string          `json:"exit_reason"`
	TotalIterations int             `json:"total_iterations"`
	TotalCommits    int             `json:"total_commits"`
	InitialTodos    []string        `json:"initial_todos,omitempty"`
	Iterations      []IterationData `json:"iterations"`
	Errors          []ErrorData     `json:"errors,omitempty"`
}

// IterationData holds data for a single iteration
type IterationData struct {
	Number         int           `json:"number"`
	StartTime      time.Time     `json:"start_time"`
	Duration       time.Duration `json:"duration_ns"`
	ExitCode       int           `json:"exit_code"`
	Commits        int           `json:"commits"`
	TodosCompleted []string      `json:"todos_completed,omitempty"`
}

// ErrorData holds data for a single error
type ErrorData struct {
	Iteration int       `json:"iteration"`
	ExitCode  int       `json:"exit_code"`
	Phase     string    `json:"phase"`
	Time      time.Time `json:"timestamp"`
}

// saveRunData saves the run data to both JSON and markdown files
func (m *Manager) saveRunData() {
	m.mu.Lock()
	data := *m.runData
	m.mu.Unlock()

	runsDir := filepath.Join(m.agentDir, "runs")

	// Save JSON
	jsonPath := filepath.Join(runsDir, m.sessionID+".json")
	if jsonBytes, err := json.MarshalIndent(data, "", "  "); err == nil {
		os.WriteFile(jsonPath, jsonBytes, 0644)
	}

	// Save markdown summary
	mdPath := filepath.Join(runsDir, m.sessionID+".md")
	md := m.generateMarkdownSummary(data)
	os.WriteFile(mdPath, []byte(md), 0644)
}

// generateMarkdownSummary creates a human-readable summary of the run
func (m *Manager) generateMarkdownSummary(data RunData) string {
	duration := data.EndTime.Sub(data.StartTime).Round(time.Second)

	md := "# Ralph Run Summary\n\n"
	md += fmt.Sprintf("**Session:** %s\n", data.SessionID)
	md += fmt.Sprintf("**Project:** %s\n", data.ProjectName)
	md += fmt.Sprintf("**Date:** %s - %s (%s)\n",
		data.StartTime.Format("2006-01-02 15:04"),
		data.EndTime.Format("15:04"),
		duration)
	md += fmt.Sprintf("**Exit Reason:** %s\n\n", data.ExitReason)

	// Results table
	md += "## Results\n\n"
	md += "| Metric | Value |\n"
	md += "|--------|-------|\n"
	md += fmt.Sprintf("| Iterations | %d |\n", data.TotalIterations)
	md += fmt.Sprintf("| Duration | %s |\n", duration)
	md += fmt.Sprintf("| Commits | %d |\n", data.TotalCommits)
	md += fmt.Sprintf("| Errors | %d |\n", len(data.Errors))
	md += "\n"

	// Tasks completed
	if len(data.Iterations) > 0 {
		var allCompleted []string
		for _, iter := range data.Iterations {
			allCompleted = append(allCompleted, iter.TodosCompleted...)
		}
		if len(allCompleted) > 0 {
			md += "## Tasks Completed\n\n"
			for i, task := range allCompleted {
				md += fmt.Sprintf("%d. [x] %s\n", i+1, task)
			}
			md += "\n"
		}
	}

	// Errors
	if len(data.Errors) > 0 {
		md += "## Errors\n\n"
		for _, err := range data.Errors {
			md += fmt.Sprintf("- Iteration %d: Exit code %d", err.Iteration, err.ExitCode)
			if err.Phase != "" {
				md += fmt.Sprintf(" during %s", err.Phase)
			}
			md += "\n"
		}
		md += "\n"
	}

	// Timeline
	md += "## Timeline\n\n"
	md += fmt.Sprintf("- %s - Session started\n", data.StartTime.Format("15:04:05"))
	for _, iter := range data.Iterations {
		endTime := iter.StartTime.Add(iter.Duration)
		md += fmt.Sprintf("- %s - Iteration %d ended (%s, %d commits)\n",
			endTime.Format("15:04:05"),
			iter.Number,
			iter.Duration.Round(time.Second),
			iter.Commits)
	}
	md += fmt.Sprintf("- %s - Session ended\n", data.EndTime.Format("15:04:05"))

	return md
}
