package github

import (
	"fmt"
	"strings"
)

// GeneratePrompt converts a GitHub issue into prompt.md content
func GeneratePrompt(issue *Issue) string {
	var sb strings.Builder

	// Header with issue number and title
	sb.WriteString(fmt.Sprintf("# Issue #%d: %s\n\n", issue.Number, issue.Title))

	// Issue body
	if issue.Body != "" {
		sb.WriteString(issue.Body)
		sb.WriteString("\n")
	}

	// Metadata section
	sb.WriteString("\n---\n")
	sb.WriteString(fmt.Sprintf("Source: %s\n", issue.URL))

	if len(issue.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(issue.Labels, ", ")))
	}

	if len(issue.Assignees) > 0 {
		sb.WriteString(fmt.Sprintf("Assignees: %s\n", strings.Join(issue.Assignees, ", ")))
	}

	if issue.Milestone != "" {
		sb.WriteString(fmt.Sprintf("Milestone: %s\n", issue.Milestone))
	}

	return sb.String()
}
