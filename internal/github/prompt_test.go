package github

import (
	"strings"
	"testing"
)

func TestGeneratePrompt(t *testing.T) {
	tests := []struct {
		name     string
		issue    *Issue
		contains []string
		excludes []string
	}{
		{
			name: "full issue",
			issue: &Issue{
				Number:    42,
				Title:     "Add new feature",
				Body:      "This is the issue description.\n\nWith multiple lines.",
				Labels:    []string{"enhancement", "help wanted"},
				URL:       "https://github.com/owner/repo/issues/42",
				Assignees: []string{"user1", "user2"},
				Milestone: "v1.0",
			},
			contains: []string{
				"# Issue #42: Add new feature",
				"This is the issue description.",
				"With multiple lines.",
				"Source: https://github.com/owner/repo/issues/42",
				"Labels: enhancement, help wanted",
				"Assignees: user1, user2",
				"Milestone: v1.0",
			},
		},
		{
			name: "minimal issue",
			issue: &Issue{
				Number: 1,
				Title:  "Bug fix",
				Body:   "",
				URL:    "https://github.com/owner/repo/issues/1",
			},
			contains: []string{
				"# Issue #1: Bug fix",
				"Source: https://github.com/owner/repo/issues/1",
			},
			excludes: []string{
				"Labels:",
				"Assignees:",
				"Milestone:",
			},
		},
		{
			name: "issue with only labels",
			issue: &Issue{
				Number: 5,
				Title:  "Documentation update",
				Body:   "Update the README",
				Labels: []string{"docs"},
				URL:    "https://github.com/owner/repo/issues/5",
			},
			contains: []string{
				"# Issue #5: Documentation update",
				"Update the README",
				"Labels: docs",
			},
			excludes: []string{
				"Assignees:",
				"Milestone:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePrompt(tt.issue)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("GeneratePrompt() missing expected string %q\nGot:\n%s", want, result)
				}
			}

			for _, excluded := range tt.excludes {
				if strings.Contains(result, excluded) {
					t.Errorf("GeneratePrompt() should not contain %q\nGot:\n%s", excluded, result)
				}
			}
		})
	}
}

func TestGeneratePrompt_Format(t *testing.T) {
	issue := &Issue{
		Number: 42,
		Title:  "Test Issue",
		Body:   "Body content",
		URL:    "https://github.com/owner/repo/issues/42",
	}

	result := GeneratePrompt(issue)

	// Check that it starts with the header
	if !strings.HasPrefix(result, "# Issue #42:") {
		t.Errorf("Prompt should start with issue header, got: %s", result[:50])
	}

	// Check that it contains the separator
	if !strings.Contains(result, "\n---\n") {
		t.Error("Prompt should contain --- separator")
	}

	// Check that Source comes after separator
	separatorIdx := strings.Index(result, "\n---\n")
	sourceIdx := strings.Index(result, "Source:")
	if sourceIdx < separatorIdx {
		t.Error("Source should come after the --- separator")
	}
}
