package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

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

	// GitHub issue options
	Issue       string // GitHub issue reference (number, URL, or owner/repo#number)
	IssueNumber int    // Parsed issue number (set internally)
	IssueTitle  string // Parsed issue title (set internally)
	IssueURL    string // Parsed issue URL (set internally)

	// Prompt options
	ScratchpadPrompt string // Custom scratchpad instructions (appended to prompt)

	// Sound options
	SoundEnabled  bool   // Play Ralph Wiggum quotes after each iteration
	SoundMute     bool   // Mute sound playback (respects RALPH_SOUND_MUTE env)
	SoundPageURL  string // URL to fetch sound clips from
	SoundPlayer   string // Preferred audio player (afplay, ffplay, mpg123, mpg321)
	SoundCacheDir string // Directory to cache sound URLs

	// Session info
	SessionID string

	// Config file path (set by --config flag)
	ConfigFile string

	// Test mode options
	TestMode     bool   // Run in test mode (mock Claude, simulate todo progress)
	TestScenario string // Test scenario: "success", "error", "partial"

	// State logging options
	StateLoggingEnabled bool // Enable automatic state logging (default: true)
	RunsRetention       int  // Number of runs to keep in runs/ (default: 50)

	// TUI options
	TUIBufferSize int // Number of lines to keep in TUI scroll buffer (default: 10000)
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
		MaxIterations *int   `yaml:"max_iterations"`
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

	Issue string `yaml:"issue"` // GitHub issue reference

	Sound struct {
		Enabled  *bool  `yaml:"enabled"`
		Mute     *bool  `yaml:"mute"`
		PageURL  string `yaml:"page_url"`
		Player   string `yaml:"player"`
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"sound"`

	ScratchpadPrompt string `yaml:"scratchpad_prompt"`

	TestMode struct {
		Enabled  *bool  `yaml:"enabled"`
		Scenario string `yaml:"scenario"`
	} `yaml:"test_mode"`

	StateLogging struct {
		Enabled       *bool `yaml:"enabled"`
		RunsRetention *int  `yaml:"runs_retention"`
	} `yaml:"state_logging"`

	TUI struct {
		BufferSize *int `yaml:"buffer_size"`
	} `yaml:"tui"`
}

// DefaultConfig returns a Config with default values matching the bash script
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	return &Config{
		PromptFile:    "./.agent/IMPLEMENTATION_PLAN.md",
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
		CodeReviewMaxIterations: 0,
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

		ScratchpadPrompt: DefaultScratchpadPrompt,

		SoundEnabled: false,
		SoundMute:     getEnvOrDefault("RALPH_SOUND_MUTE", "") == "1",
		SoundPageURL:  getEnvOrDefault("RALPH_SOUND_PAGE_URL", "https://andrewziola.com/xoom/wiggum/"),
		SoundPlayer:   getEnvOrDefault("RALPH_SOUND_PLAYER", ""),
		SoundCacheDir: getSoundCacheDir(),

		SessionID: uuid.New().String(),

		TestMode:     false,
		TestScenario: "success",

		StateLoggingEnabled: true,
		RunsRetention:       50,

		TUIBufferSize: 10000,
	}
}

// getSoundCacheDir returns the cache directory for sound files
func getSoundCacheDir() string {
	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = "/tmp"
	}
	return filepath.Join(cacheDir, "ralph")
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

// DefaultScratchpadPrompt is the default prompt appended to instructions
const DefaultScratchpadPrompt = `Use the {{.AgentDir}} directory as a scratchpad for your work.

## CRITICAL: One Task Per Session

You are running in a loop. Each session should complete ONE small task, then EXIT.
This prevents context rot and keeps each session focused.

Examples of "one task":
- Create or update the implementation plan in TODO.md
- Implement a single function or fix a single bug
- Add tests for one component
- Fix one failing test

When your single task is complete: COMMIT and EXIT. The loop will start a fresh session.
Do NOT try to complete multiple tasks in one session. Do NOT continue "just to finish one more thing."

## Directory Structure
- {{.AgentDir}}/IMPLEMENTATION_PLAN.md - The main implementation plan (your input)
- {{.AgentDir}}/TODO.md - Granular task checkboxes (update as you work)
- {{.AgentDir}}/progress.md - Progress log (Ralph appends automatically on task completion)
- {{.AgentDir}}/guardrails.md - Lessons learned / "Signs" (you can read and write)
- {{.AgentDir}}/activity.log - Activity timing log (Ralph writes automatically)
- {{.AgentDir}}/errors.log - Error log (Ralph writes automatically)
- {{.AgentDir}}/runs/ - Historical run data (Ralph writes automatically)

## Task Tracking
- Keep track of your current status in {{.AgentDir}}/TODO.md using checkboxes (- [ ] for pending, - [-] for in-progress, - [x] for done).
- Check off items when completed. Only work on a single item at a time.
- If a bug or new behavior comes up during implementation, add it to the todo list.

## Guardrails (Lessons Learned)
- Use {{.AgentDir}}/guardrails.md to record important lessons learned during implementation.
- Check this file at the start of each session for context from previous runs.
- Add notes about gotchas, workarounds, or important decisions.

## Starting a Session
1. Read TODO.md to find the next pending task (or create the plan if none exists)
2. Check guardrails.md for lessons from previous sessions
3. Pick ONE task to complete this session

## Testing
- If the project has tests, always run them before and after making your changes.
- Do not accept any regressions. Always add new tests for new functionality.

## Completing Your Single Task
1. Finish your one task
2. Mark it complete in TODO.md (- [x])
3. Commit your changes
4. EXIT immediately - do not start another task

The loop handles everything else. Trust the loop.`

// ScratchpadInstructions returns the instructions appended to prompts
// Supports {{.AgentDir}} template substitution in the prompt
func (c *Config) ScratchpadInstructions() string {
	prompt := c.ScratchpadPrompt
	if prompt == "" {
		prompt = DefaultScratchpadPrompt
	}
	prompt = strings.ReplaceAll(prompt, "{{.AgentDir}}", c.AgentDir)
	return "\n\n" + prompt
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
	if yc.CodeReview.MaxIterations != nil {
		c.CodeReviewMaxIterations = *yc.CodeReview.MaxIterations
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

	// GitHub issue options
	if yc.Issue != "" {
		c.Issue = yc.Issue
	}

	// Sound options
	if yc.Sound.Enabled != nil {
		c.SoundEnabled = *yc.Sound.Enabled
	}
	if yc.Sound.Mute != nil {
		c.SoundMute = *yc.Sound.Mute
	}
	if yc.Sound.PageURL != "" {
		c.SoundPageURL = yc.Sound.PageURL
	}
	if yc.Sound.Player != "" {
		c.SoundPlayer = yc.Sound.Player
	}
	if yc.Sound.CacheDir != "" {
		c.SoundCacheDir = yc.Sound.CacheDir
	}

	// Prompt options
	if yc.ScratchpadPrompt != "" {
		c.ScratchpadPrompt = yc.ScratchpadPrompt
	}

	// Test mode options
	if yc.TestMode.Enabled != nil {
		c.TestMode = *yc.TestMode.Enabled
	}
	if yc.TestMode.Scenario != "" {
		c.TestScenario = yc.TestMode.Scenario
	}

	// State logging options
	if yc.StateLogging.Enabled != nil {
		c.StateLoggingEnabled = *yc.StateLogging.Enabled
	}
	if yc.StateLogging.RunsRetention != nil {
		c.RunsRetention = *yc.StateLogging.RunsRetention
	}

	// TUI options
	if yc.TUI.BufferSize != nil {
		c.TUIBufferSize = *yc.TUI.BufferSize
	}

	return nil
}

// SoundConfig holds configuration for Ralph sound playback
type SoundConfig struct {
	Enabled  bool
	Mute     bool
	PageURL  string
	Player   string
	CacheDir string
}

// GetSoundConfig returns the sound configuration
func (c *Config) GetSoundConfig() SoundConfig {
	return SoundConfig{
		Enabled:  c.SoundEnabled,
		Mute:     c.SoundMute,
		PageURL:  c.SoundPageURL,
		Player:   c.SoundPlayer,
		CacheDir: c.SoundCacheDir,
	}
}
