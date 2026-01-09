package slack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Notifier handles high-level Slack notifications for ralph sessions
type Notifier struct {
	client       *Client
	enabled      bool
	threadTS     string // Thread timestamp for replies
	channel      string
	notifyUsers  []string
	projectName  string
	githubURL    string
	tmuxSession  string
	sessionID    string
	startTime    time.Time
	maxIters     int
	maxTime      int
}

// NotifierConfig holds configuration for the notifier
type NotifierConfig struct {
	Enabled       bool
	WebhookURL    string
	BotToken      string
	Channel       string
	NotifyUsers   []string
	ProjectName   string
	SessionID     string
	MaxIterations int
	MaxTime       int
}

// NewNotifier creates a new Slack notifier
func NewNotifier(cfg NotifierConfig) *Notifier {
	n := &Notifier{
		enabled:     cfg.Enabled,
		channel:     cfg.Channel,
		notifyUsers: cfg.NotifyUsers,
		projectName: cfg.ProjectName,
		sessionID:   cfg.SessionID,
		startTime:   time.Now(),
		maxIters:    cfg.MaxIterations,
		maxTime:     cfg.MaxTime,
	}

	if cfg.Enabled {
		n.client = NewClient(cfg.WebhookURL, cfg.BotToken, cfg.Channel)
		n.githubURL = getGitHubURL()
		n.tmuxSession = getTmuxSession()
	}

	return n
}

// IsEnabled returns whether Slack notifications are enabled
func (n *Notifier) IsEnabled() bool {
	return n.enabled && n.client != nil && n.client.IsConfigured()
}

// SessionStart sends the initial session start message
func (n *Notifier) SessionStart(ctx context.Context) error {
	if !n.IsEnabled() {
		return nil
	}

	msg := FormatSessionStart(SessionStartInfo{
		ProjectName:   n.projectName,
		GitHubURL:     n.githubURL,
		TmuxSession:   n.tmuxSession,
		StartTime:     n.startTime,
		SessionID:     n.sessionID,
		MaxIterations: n.maxIters,
		MaxTime:       n.maxTime,
	})

	// Try to post via bot API first (to get thread TS)
	if n.client.botToken != "" && n.channel != "" {
		req := &ChatPostMessageRequest{
			Channel: n.channel,
			Text:    msg.Text,
		}

		err := n.client.PostWithRetry(ctx, func() error {
			resp, err := n.client.PostMessage(ctx, req)
			if err != nil {
				return err
			}
			n.threadTS = resp.TS
			return nil
		})
		if err != nil {
			// Disable further notifications if initial post fails
			n.enabled = false
			return fmt.Errorf("session start notification: %w", err)
		}
		return nil
	}

	// Fall back to webhook (no threading support)
	err := n.client.PostWithRetry(ctx, func() error {
		return n.client.PostWebhook(ctx, msg)
	})
	if err != nil {
		n.enabled = false
		return fmt.Errorf("session start notification: %w", err)
	}

	return nil
}

// TodoStarted sends a notification when a todo item is started
func (n *Notifier) TodoStarted(ctx context.Context, todoText string, currentIndex, total, iteration, completed int) error {
	if !n.IsEnabled() {
		return nil
	}

	// Thread replies require bot token
	if n.client.botToken == "" || n.threadTS == "" {
		return nil
	}

	msg := FormatTodoStarted(TodoStartedInfo{
		TodoText:       todoText,
		CurrentIndex:   currentIndex,
		TotalCount:     total,
		Iteration:      iteration,
		CompletedCount: completed,
	})

	msg.Channel = n.channel
	msg.ThreadTS = n.threadTS

	return n.client.PostWithRetry(ctx, func() error {
		_, err := n.client.PostMessage(ctx, msg)
		return err
	})
}

// TodoCompleted sends a notification when todo items are completed
func (n *Notifier) TodoCompleted(ctx context.Context, todoText string, completed, total, iteration, commits int, iterDuration time.Duration) error {
	if !n.IsEnabled() {
		return nil
	}

	// Thread replies require bot token
	if n.client.botToken == "" || n.threadTS == "" {
		return nil
	}

	msg := FormatTodoCompleted(TodoCompletedInfo{
		TodoText:          todoText,
		CompletedCount:    completed,
		TotalCount:        total,
		Iteration:         iteration,
		Commits:           commits,
		IterationDuration: iterDuration,
	})

	msg.Channel = n.channel
	msg.ThreadTS = n.threadTS

	return n.client.PostWithRetry(ctx, func() error {
		_, err := n.client.PostMessage(ctx, msg)
		return err
	})
}

// CodeReviewStarted sends a notification when code review phase starts
func (n *Notifier) CodeReviewStarted(ctx context.Context, iteration, maxIterations int) error {
	if !n.IsEnabled() {
		return nil
	}

	// Thread replies require bot token
	if n.client.botToken == "" || n.threadTS == "" {
		return nil
	}

	msg := FormatCodeReviewStarted(CodeReviewStartedInfo{
		Iteration:     iteration,
		MaxIterations: maxIterations,
	})

	msg.Channel = n.channel
	msg.ThreadTS = n.threadTS

	return n.client.PostWithRetry(ctx, func() error {
		_, err := n.client.PostMessage(ctx, msg)
		return err
	})
}

// CodeReviewComplete sends a notification when code review phase completes
func (n *Notifier) CodeReviewComplete(ctx context.Context, iterations, issuesFound, issuesFixed int, duration time.Duration) error {
	if !n.IsEnabled() {
		return nil
	}

	// Thread replies require bot token
	if n.client.botToken == "" || n.threadTS == "" {
		return nil
	}

	msg := FormatCodeReviewComplete(CodeReviewCompleteInfo{
		Iterations:  iterations,
		IssuesFound: issuesFound,
		IssuesFixed: issuesFixed,
		Duration:    duration,
	})

	msg.Channel = n.channel
	msg.ThreadTS = n.threadTS

	return n.client.PostWithRetry(ctx, func() error {
		_, err := n.client.PostMessage(ctx, msg)
		return err
	})
}

// SessionEnd sends the final session summary
func (n *Notifier) SessionEnd(ctx context.Context, summary SessionSummary) error {
	if !n.IsEnabled() {
		return nil
	}

	summary.NotifyUsers = n.notifyUsers
	msg := FormatSessionEnd(summary)

	// Thread reply if we have thread TS
	if n.client.botToken != "" && n.threadTS != "" {
		msg.Channel = n.channel
		msg.ThreadTS = n.threadTS

		return n.client.PostWithRetry(ctx, func() error {
			_, err := n.client.PostMessage(ctx, msg)
			return err
		})
	}

	// Fall back to webhook
	webhookMsg := &WebhookMessage{
		Text:    msg.Text,
		Channel: n.channel,
	}

	return n.client.PostWithRetry(ctx, func() error {
		return n.client.PostWebhook(ctx, webhookMsg)
	})
}

// getGitHubURL extracts the GitHub URL from git remote
func getGitHubURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))

	// Convert SSH URLs to HTTPS
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)
	}

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Only return GitHub URLs
	if strings.Contains(url, "github.com") {
		return url
	}

	return ""
}

// getTmuxSession extracts the tmux session name from the environment
func getTmuxSession() string {
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return ""
	}

	// Try to get session name from tmux command
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
