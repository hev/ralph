package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrorEntry represents a single error log entry
type ErrorEntry struct {
	Iteration int
	ExitCode  int
	Phase     string
	Context   string
	Output    string
	Time      time.Time
}

// appendError appends an error entry to errors.log
func (m *Manager) appendError(entry ErrorEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.agentDir, "errors.log")

	// Check if file exists, if not create with header
	if _, err := os.Stat(path); os.IsNotExist(err) {
		header := "# Error Log\n\n"
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
	provider := m.provider
	if provider == "" {
		provider = "claude"
	}
	content := fmt.Sprintf("## %s - %s Exit Error\n", timestamp, provider)
	content += fmt.Sprintf("**Iteration:** %d\n", entry.Iteration)
	content += fmt.Sprintf("**Exit Code:** %d\n", entry.ExitCode)
	if entry.Phase != "" {
		content += fmt.Sprintf("**Phase:** %s\n", entry.Phase)
	}
	if entry.Context != "" {
		content += fmt.Sprintf("**Context:** %s\n", entry.Context)
	}
	if entry.Output != "" {
		content += fmt.Sprintf("**Previous Output:** %s\n", truncateOutput(entry.Output, 500))
	}
	content += "\n---\n\n"

	f.WriteString(content)
}

// truncateOutput truncates output to a maximum length
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-maxLen:]
}
