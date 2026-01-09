package slack

import (
	"fmt"
	"strings"
	"time"
)

// SessionStartInfo contains information for the session start message
type SessionStartInfo struct {
	ProjectName   string
	GitHubURL     string
	TmuxSession   string
	StartTime     time.Time
	SessionID     string
	MaxIterations int
	MaxTime       int
}

// TodoCompletedInfo contains information for todo completion messages
type TodoCompletedInfo struct {
	TodoText          string
	CompletedCount    int
	TotalCount        int
	Iteration         int
	Commits           int
	IterationDuration time.Duration
}

// TodoStartedInfo contains information for todo started messages
type TodoStartedInfo struct {
	TodoText       string
	CurrentIndex   int
	TotalCount     int
	Iteration      int
	CompletedCount int
}

// SessionSummary contains information for the session end message
type SessionSummary struct {
	Iterations  int
	Duration    time.Duration
	Commits     int
	TodosDone   int
	TodosTotal  int
	ExitReason  string
	NotifyUsers []string
}

// CodeReviewStartedInfo contains information for code review start messages
type CodeReviewStartedInfo struct {
	Iteration     int
	MaxIterations int
}

// CodeReviewCompleteInfo contains information for code review complete messages
type CodeReviewCompleteInfo struct {
	Iterations    int
	IssuesFound   int
	IssuesFixed   int
	Duration      time.Duration
}

// FormatSessionStart creates the session start message
func FormatSessionStart(info SessionStartInfo) *WebhookMessage {
	var lines []string
	lines = append(lines, "*Ralph session started*")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("*Project:* %s", info.ProjectName))

	if info.GitHubURL != "" {
		lines = append(lines, fmt.Sprintf("*GitHub:* <%s|%s>", info.GitHubURL, info.GitHubURL))
	}

	if info.TmuxSession != "" {
		lines = append(lines, fmt.Sprintf("*tmux:* `%s`", info.TmuxSession))
	}

	lines = append(lines, fmt.Sprintf("*Started:* %s", info.StartTime.Format("2006-01-02 15:04:05 UTC")))
	lines = append(lines, fmt.Sprintf("*Session:* `%s`", truncateSessionID(info.SessionID)))
	lines = append(lines, "")

	// Format limits
	var limits []string
	if info.MaxIterations > 0 {
		limits = append(limits, fmt.Sprintf("%d iterations", info.MaxIterations))
	} else {
		limits = append(limits, "unlimited iterations")
	}
	if info.MaxTime > 0 {
		limits = append(limits, fmt.Sprintf("%ds max", info.MaxTime))
	} else {
		limits = append(limits, "unlimited time")
	}
	lines = append(lines, fmt.Sprintf("*Limits:* %s", strings.Join(limits, " / ")))

	return &WebhookMessage{
		Text: strings.Join(lines, "\n"),
	}
}

// FormatTodoStarted creates a todo started update message
func FormatTodoStarted(info TodoStartedInfo) *ChatPostMessageRequest {
	var lines []string
	lines = append(lines, fmt.Sprintf("*Working on item %d of %d*", info.CurrentIndex, info.TotalCount))
	lines = append(lines, fmt.Sprintf("_%s_", info.TodoText))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Iteration: %d | Completed: %d/%d",
		info.Iteration, info.CompletedCount, info.TotalCount))

	return &ChatPostMessageRequest{
		Text: strings.Join(lines, "\n"),
	}
}

// FormatTodoCompleted creates a todo completion update message
func FormatTodoCompleted(info TodoCompletedInfo) *ChatPostMessageRequest {
	var lines []string
	lines = append(lines, fmt.Sprintf("*Todo completed (%d/%d)*", info.CompletedCount, info.TotalCount))
	lines = append(lines, fmt.Sprintf("_%s_", info.TodoText))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Iteration: %d | Commits: %d | Duration: %s",
		info.Iteration, info.Commits, formatDuration(info.IterationDuration)))

	return &ChatPostMessageRequest{
		Text: strings.Join(lines, "\n"),
	}
}

// FormatSessionEnd creates the session end message
func FormatSessionEnd(summary SessionSummary) *ChatPostMessageRequest {
	var lines []string
	lines = append(lines, "*Ralph session complete*")
	lines = append(lines, "")
	lines = append(lines, "*Summary:*")
	lines = append(lines, fmt.Sprintf("• Iterations: %d", summary.Iterations))
	lines = append(lines, fmt.Sprintf("• Duration: %s", formatDuration(summary.Duration)))
	lines = append(lines, fmt.Sprintf("• Commits: %d", summary.Commits))

	if summary.TodosTotal > 0 {
		pct := float64(summary.TodosDone) / float64(summary.TodosTotal) * 100
		lines = append(lines, fmt.Sprintf("• Todos: %d/%d complete (%.0f%%)", summary.TodosDone, summary.TodosTotal, pct))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("*Exit reason:* %s", summary.ExitReason))

	// Add user mentions if configured
	if len(summary.NotifyUsers) > 0 {
		var mentions []string
		for _, user := range summary.NotifyUsers {
			mentions = append(mentions, fmt.Sprintf("<@%s>", user))
		}
		lines = append(lines, "")
		lines = append(lines, "cc: "+strings.Join(mentions, " "))
	}

	return &ChatPostMessageRequest{
		Text: strings.Join(lines, "\n"),
	}
}

// FormatCodeReviewStarted creates a code review started message
func FormatCodeReviewStarted(info CodeReviewStartedInfo) *ChatPostMessageRequest {
	var lines []string
	lines = append(lines, "*Code review phase started*")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Review iteration: %d of %d", info.Iteration, info.MaxIterations))

	return &ChatPostMessageRequest{
		Text: strings.Join(lines, "\n"),
	}
}

// FormatCodeReviewComplete creates a code review complete message
func FormatCodeReviewComplete(info CodeReviewCompleteInfo) *ChatPostMessageRequest {
	var lines []string
	lines = append(lines, "*Code review complete*")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("• Iterations: %d", info.Iterations))
	lines = append(lines, fmt.Sprintf("• Issues found: %d", info.IssuesFound))
	lines = append(lines, fmt.Sprintf("• Issues fixed: %d", info.IssuesFixed))
	lines = append(lines, fmt.Sprintf("• Duration: %s", formatDuration(info.Duration)))

	return &ChatPostMessageRequest{
		Text: strings.Join(lines, "\n"),
	}
}

// truncateSessionID shortens a UUID for display
func truncateSessionID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
