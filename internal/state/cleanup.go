package state

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cleanupOldRuns removes old run files beyond the retention limit
func (m *Manager) cleanupOldRuns() {
	runsDir := filepath.Join(m.agentDir, "runs")

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return
	}

	// Group files by session ID (each session has .json and .md)
	sessions := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Extract session ID by removing extension
		sessionID := strings.TrimSuffix(name, filepath.Ext(name))
		sessions[sessionID] = append(sessions[sessionID], name)
	}

	// If we have more sessions than retention limit, remove oldest
	if len(sessions) <= m.runsRetention {
		return
	}

	// Get session IDs sorted by file modification time
	type sessionInfo struct {
		id      string
		modTime int64
	}
	var sessionList []sessionInfo

	for sessionID := range sessions {
		// Use the JSON file's mod time as the session time
		jsonPath := filepath.Join(runsDir, sessionID+".json")
		info, err := os.Stat(jsonPath)
		if err != nil {
			// Try the md file
			mdPath := filepath.Join(runsDir, sessionID+".md")
			info, err = os.Stat(mdPath)
			if err != nil {
				continue
			}
		}
		sessionList = append(sessionList, sessionInfo{
			id:      sessionID,
			modTime: info.ModTime().Unix(),
		})
	}

	// Sort by modification time (oldest first)
	sort.Slice(sessionList, func(i, j int) bool {
		return sessionList[i].modTime < sessionList[j].modTime
	})

	// Remove oldest sessions until we're at retention limit
	toRemove := len(sessionList) - m.runsRetention
	for i := 0; i < toRemove && i < len(sessionList); i++ {
		sessionID := sessionList[i].id
		for _, filename := range sessions[sessionID] {
			os.Remove(filepath.Join(runsDir, filename))
		}
	}
}
