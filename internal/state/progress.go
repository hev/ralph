package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProgressEntry represents a single progress log entry
type ProgressEntry struct {
	Task     string
	Duration time.Duration
	Commits  int
	Notes    string
	Time     time.Time
}

// appendProgress appends a progress entry to progress.md
func (m *Manager) appendProgress(entry ProgressEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.agentDir, "progress.md")

	// Check if file exists, if not create with header
	if _, err := os.Stat(path); os.IsNotExist(err) {
		header := "# Progress Log\n\n"
		if err := os.WriteFile(path, []byte(header), 0644); err != nil {
			return
		}
	}

	// Append the entry
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	content := fmt.Sprintf("## %s - Task Completed\n", timestamp)
	content += fmt.Sprintf("**Task:** %s\n", entry.Task)
	content += fmt.Sprintf("**Duration:** %s\n", entry.Duration.Round(time.Second))
	content += fmt.Sprintf("**Commits:** %d\n", entry.Commits)
	if entry.Notes != "" {
		content += fmt.Sprintf("**Notes:** %s\n", entry.Notes)
	}
	content += "\n---\n\n"

	f.WriteString(content)
}
