package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// logActivity writes a structured log entry to activity.log
func (m *Manager) logActivity(event string, fields map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.agentDir, "activity.log")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Build the log line with consistent field ordering
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", timestamp))
	parts = append(parts, event)

	// Sort keys for consistent output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, fields[k]))
	}

	line := strings.Join(parts, " ") + "\n"
	f.WriteString(line)
}
