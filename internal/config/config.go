package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
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

	// Behavior options
	StopOnCompletion bool // Exit when all todos are complete

	// Code review options
	CodeReviewEnabled       bool   // Run code review phase after todos complete
	CodeReviewMaxIterations int    // Max iterations for code review phase
	CodeReviewPrompt        string // Prompt to use for code review phase

	// Cleanup options
	CleanupEnabled  bool     // Run cleanup phase after code review
	CleanupPatterns []string // Glob patterns for files to clean up

	// Worktree options
	WorktreeEnabled      bool   // Run in a git worktree
	WorktreeBranch       string // Branch name for worktree (empty = auto-generate)
	WorktreeBaseDir      string // Where to create worktrees
	WorktreeBranchPrefix string // Prefix for auto-generated branch names
	WorktreeCleanup      bool   // Delete worktree on completion

	// Model options
	Model           string // Model to use for main phase (e.g., "sonnet", "opus", "haiku")
	CodeReviewModel string // Model to use for code review phase (defaults to Model)
	CleanupModel    string // Model to use for cleanup phase (defaults to Model)

	// PR options
	PREnabled bool   // Create a PR when the loop completes
	PRTitle   string // Custom title for the PR (empty = auto-generate)
	PRBase    string // Base branch for the PR (empty = default branch)

	// Session info
	SessionID string

	// Config file path (set by --config flag)
	ConfigFile string
}

// yamlConfig represents the YAML file structure
type yamlConfig struct {
	Prompt        string `yaml:"prompt"`
	MaxIterations int    `yaml:"max_iterations"`
	MaxTime       int    `yaml:"max_time"`
	AgentDir      string `yaml:"agent_dir"`
	Cooldown      int    `yaml:"cooldown"`
	Verbose       *bool  `yaml:"verbose"`
	DryRun        *bool  `yaml:"dry_run"`
	Model         string `yaml:"model"`

	OTEL struct {
		Enabled       *bool  `yaml:"enabled"`
		Endpoint      string `yaml:"endpoint"`
		MetricsPrefix string `yaml:"metrics_prefix"`
		ProjectName   string `yaml:"project_name"`
	} `yaml:"otel"`

	Slack struct {
		Enabled     *bool  `yaml:"enabled"`
		WebhookURL  string `yaml:"webhook_url"`
		BotToken    string `yaml:"bot_token"`
		Channel     string `yaml:"channel"`
		NotifyUsers string `yaml:"notify_users"`
	} `yaml:"slack"`

	StopOnCompletion *bool `yaml:"stop_on_completion"`

	CodeReview struct {
		Enabled       *bool  `yaml:"enabled"`
		MaxIterations int    `yaml:"max_iterations"`
		Prompt        string `yaml:"prompt"`
		Model         string `yaml:"model"`
	} `yaml:"code_review"`

	Cleanup struct {
		Enabled  *bool    `yaml:"enabled"`
		Patterns []string `yaml:"patterns"`
		Model    string   `yaml:"model"`
	} `yaml:"cleanup"`

	Worktree struct {
		Enabled      *bool  `yaml:"enabled"`
		BaseDir      string `yaml:"base_dir"`
		BranchPrefix string `yaml:"branch_prefix"`
		Cleanup      *bool  `yaml:"cleanup"`
	} `yaml:"worktree"`

	PR struct {
		Enabled *bool  `yaml:"enabled"`
		Title   string `yaml:"title"`
		Base    string `yaml:"base"`
	} `yaml:"pr"`
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
		Cooldown:      0,
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

		CodeReviewEnabled:       false,
		CodeReviewMaxIterations: 3,
		CodeReviewPrompt:        "",

		CleanupEnabled: false,
		CleanupPatterns: []string{
			"**/*_test_*.go",
			"**/*.test.js",
			"**/*.test.ts",
			"**/*.spec.js",
			"**/*.spec.ts",
			"**/test_*.py",
			"**/*_test.py",
			".agent/TODO.md",
		},

		WorktreeEnabled:      false,
		WorktreeBranch:       "",
		WorktreeBaseDir:      "/tmp/ralph-worktrees",
		WorktreeBranchPrefix: "ralph/",
		WorktreeCleanup:      true,

		PREnabled: false,
		PRTitle:   "",
		PRBase:    "",

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

// CodeReviewInstructions returns the default code review prompt
func (c *Config) CodeReviewInstructions() string {
	if c.CodeReviewPrompt != "" {
		return c.CodeReviewPrompt
	}
	return `Review the code changes made in this session. Look for:
1. Bugs or logic errors
2. Security vulnerabilities
3. Performance issues
4. Code style inconsistencies
5. Missing error handling
6. Incomplete implementations

For each issue found:
- Create a TODO item in ` + c.AgentDir + `/TODO.md describing the fix needed
- Use checkboxes (- [ ] for pending, - [x] for done)

If no issues are found, add a single TODO item: "- [x] Code review complete - no issues found"

Make a commit and push your changes after every single file edit.`
}

// FindConfigFile searches for a config file in standard locations
// Returns the path if found, empty string otherwise
// Deprecated: Use FindConfigFiles() for two-tier config support
func FindConfigFile() string {
	// Check current directory first
	if _, err := os.Stat("ralph.yaml"); err == nil {
		return "ralph.yaml"
	}

	// Check user config directory
	home, err := os.UserHomeDir()
	if err == nil {
		userConfig := filepath.Join(home, ".config", "ralph", "ralph.yaml")
		if _, err := os.Stat(userConfig); err == nil {
			return userConfig
		}
	}

	return ""
}

// FindConfigFiles returns paths to config files in load order (global first, local second)
// Global config: ~/.config/ralph/ralph.yaml
// Local config: ./ralph.yaml
// Local values override global values when both are present
func FindConfigFiles() []string {
	var paths []string

	// Global config first
	home, err := os.UserHomeDir()
	if err == nil {
		globalConfig := filepath.Join(home, ".config", "ralph", "ralph.yaml")
		if _, err := os.Stat(globalConfig); err == nil {
			paths = append(paths, globalConfig)
		}
	}

	// Local config second (overrides global)
	if _, err := os.Stat("ralph.yaml"); err == nil {
		paths = append(paths, "ralph.yaml")
	}

	return paths
}

// LoadFromFile loads configuration from a YAML file
// Only non-zero values in the file will override the current config
func (c *Config) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return err
	}

	// Apply non-zero values from YAML to config
	if yc.Prompt != "" {
		c.PromptFile = yc.Prompt
	}
	if yc.MaxIterations != 0 {
		c.MaxIterations = yc.MaxIterations
	}
	if yc.MaxTime != 0 {
		c.MaxTime = yc.MaxTime
	}
	if yc.AgentDir != "" {
		c.AgentDir = yc.AgentDir
	}
	if yc.Cooldown != 0 {
		c.Cooldown = yc.Cooldown
	}
	if yc.Verbose != nil {
		c.Verbose = *yc.Verbose
	}
	if yc.DryRun != nil {
		c.DryRun = *yc.DryRun
	}
	if yc.Model != "" {
		c.Model = yc.Model
	}

	// OTEL options
	if yc.OTEL.Enabled != nil {
		c.OTELEnabled = *yc.OTEL.Enabled
	}
	if yc.OTEL.Endpoint != "" {
		c.OTELEndpoint = yc.OTEL.Endpoint
	}
	if yc.OTEL.MetricsPrefix != "" {
		c.MetricsPrefix = yc.OTEL.MetricsPrefix
	}
	if yc.OTEL.ProjectName != "" {
		c.ProjectName = yc.OTEL.ProjectName
	}

	// Slack options
	if yc.Slack.Enabled != nil {
		c.SlackEnabled = *yc.Slack.Enabled
	}
	if yc.Slack.WebhookURL != "" {
		c.SlackWebhookURL = yc.Slack.WebhookURL
	}
	if yc.Slack.BotToken != "" {
		c.SlackBotToken = yc.Slack.BotToken
	}
	if yc.Slack.Channel != "" {
		c.SlackChannel = yc.Slack.Channel
	}
	if yc.Slack.NotifyUsers != "" {
		c.SlackNotifyUsers = yc.Slack.NotifyUsers
	}

	// Behavior options
	if yc.StopOnCompletion != nil {
		c.StopOnCompletion = *yc.StopOnCompletion
	}

	// Code review options
	if yc.CodeReview.Enabled != nil {
		c.CodeReviewEnabled = *yc.CodeReview.Enabled
	}
	if yc.CodeReview.MaxIterations != 0 {
		c.CodeReviewMaxIterations = yc.CodeReview.MaxIterations
	}
	if yc.CodeReview.Prompt != "" {
		c.CodeReviewPrompt = yc.CodeReview.Prompt
	}
	if yc.CodeReview.Model != "" {
		c.CodeReviewModel = yc.CodeReview.Model
	}

	// Cleanup options
	if yc.Cleanup.Enabled != nil {
		c.CleanupEnabled = *yc.Cleanup.Enabled
	}
	if len(yc.Cleanup.Patterns) > 0 {
		c.CleanupPatterns = yc.Cleanup.Patterns
	}
	if yc.Cleanup.Model != "" {
		c.CleanupModel = yc.Cleanup.Model
	}

	// Worktree options
	if yc.Worktree.Enabled != nil {
		c.WorktreeEnabled = *yc.Worktree.Enabled
	}
	if yc.Worktree.BaseDir != "" {
		c.WorktreeBaseDir = yc.Worktree.BaseDir
	}
	if yc.Worktree.BranchPrefix != "" {
		c.WorktreeBranchPrefix = yc.Worktree.BranchPrefix
	}
	if yc.Worktree.Cleanup != nil {
		c.WorktreeCleanup = *yc.Worktree.Cleanup
	}

	// PR options
	if yc.PR.Enabled != nil {
		c.PREnabled = *yc.PR.Enabled
	}
	if yc.PR.Title != "" {
		c.PRTitle = yc.PR.Title
	}
	if yc.PR.Base != "" {
		c.PRBase = yc.PR.Base
	}

	return nil
}
