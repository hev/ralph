package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const Version = "0.1.0"

// Config holds all configuration for ralph
type Config struct {
	// Core options (matching bash version)
	PromptFile    string
	MaxIterations int
	MaxTime       int
	AgentDir      string
	Cooldown      int
	Verbose       bool
	DryRun        bool

	// OTEL options
	OTELEnabled   bool
	OTELEndpoint  string
	MetricsPrefix string
	ProjectName   string

	// Slack options
	SlackEnabled     bool
	SlackWebhookURL  string
	SlackChannel     string
	SlackNotifyUsers string // Comma-separated user IDs
	SlackBotToken    string

	// Session info
	SessionID string
}

// DefaultConfig returns a Config with default values matching the bash script
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	return &Config{
		PromptFile:    "./prompt.md",
		MaxIterations: 0,
		MaxTime:       0,
		AgentDir:      "./.agent",
		Cooldown:      1,
		Verbose:       true,
		DryRun:        false,

		OTELEnabled:   false,
		OTELEndpoint:  "localhost:4317",
		MetricsPrefix: "ralph",
		ProjectName:   projectName,

		SlackEnabled:     false,
		SlackWebhookURL:  getEnvOrDefault("RALPH_SLACK_WEBHOOK_URL", ""),
		SlackChannel:     getEnvOrDefault("RALPH_SLACK_CHANNEL", ""),
		SlackNotifyUsers: getEnvOrDefault("RALPH_SLACK_NOTIFY_USERS", ""),
		SlackBotToken:    getEnvOrDefault("RALPH_SLACK_BOT_TOKEN", ""),

		SessionID: uuid.New().String(),
	}
}

// GetSlackNotifyUsers returns the notify users as a slice
func (c *Config) GetSlackNotifyUsers() []string {
	if c.SlackNotifyUsers == "" {
		return nil
	}
	users := strings.Split(c.SlackNotifyUsers, ",")
	var result []string
	for _, u := range users {
		u = strings.TrimSpace(u)
		if u != "" {
			result = append(result, u)
		}
	}
	return result
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ScratchpadInstructions returns the instructions appended to prompts
func (c *Config) ScratchpadInstructions() string {
	return "\n\nUse the " + c.AgentDir + " directory as a scratchpad for your work. Keep track of your current status in " + c.AgentDir + "/TODO.md using checkboxes (- [ ] for pending, - [x] for done). Check off items when completed. Only work on a single item at a time and end your session when complete. Make a commit and push your changes after every single file edit."
}
