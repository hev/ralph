package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/runner"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var cfg = config.DefaultConfig()

var rootCmd = &cobra.Command{
	Use:   "ralph",
	Short: "Run Claude in a loop",
	Long: fmt.Sprintf(`ralph v%s - Run Claude in a loop

"I'm in danger!" - Ralph Wiggum`, config.Version),
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("ralph v%s\n", config.Version)
			return nil
		}
		return runner.Run(cfg)
	},
}

var showVersion bool
var configFile string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ralph v%s\n", config.Version)
	},
}

func init() {
	// Core options (matching bash version)
	rootCmd.Flags().StringVarP(&cfg.PromptFile, "prompt", "p", cfg.PromptFile, "Path to prompt file")
	rootCmd.Flags().IntVarP(&cfg.MaxIterations, "max-iterations", "n", cfg.MaxIterations, "Max loop iterations (0 = unlimited)")
	rootCmd.Flags().IntVarP(&cfg.MaxTime, "max-time", "t", cfg.MaxTime, "Max total runtime in seconds (0 = unlimited)")
	rootCmd.Flags().StringVarP(&cfg.AgentDir, "agent-dir", "d", cfg.AgentDir, "Scratchpad directory")
	rootCmd.Flags().IntVarP(&cfg.Cooldown, "cooldown", "c", cfg.Cooldown, "Delay between iterations")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "", true, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "quiet", "q", false, "Disable verbose output")
	rootCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Show what would run without executing")

	// OTEL options
	rootCmd.Flags().BoolVar(&cfg.OTELEnabled, "otel-enabled", cfg.OTELEnabled, "Enable metrics export")
	rootCmd.Flags().StringVar(&cfg.OTELEndpoint, "otel-endpoint", cfg.OTELEndpoint, "OTLP endpoint")
	rootCmd.Flags().StringVar(&cfg.MetricsPrefix, "metrics-prefix", cfg.MetricsPrefix, "Metric name prefix")
	rootCmd.Flags().StringVar(&cfg.ProjectName, "project-name", cfg.ProjectName, "Override project label")

	// Slack options
	rootCmd.Flags().BoolVar(&cfg.SlackEnabled, "slack-enabled", cfg.SlackEnabled, "Enable Slack notifications")
	rootCmd.Flags().StringVar(&cfg.SlackWebhookURL, "slack-webhook-url", cfg.SlackWebhookURL, "Slack webhook URL")
	rootCmd.Flags().StringVar(&cfg.SlackChannel, "slack-channel", cfg.SlackChannel, "Slack channel ID")
	rootCmd.Flags().StringVar(&cfg.SlackNotifyUsers, "slack-notify-users", cfg.SlackNotifyUsers, "Comma-separated Slack user IDs to @mention on completion")
	rootCmd.Flags().StringVar(&cfg.SlackBotToken, "slack-bot-token", cfg.SlackBotToken, "Slack bot token for thread replies")

	// Behavior options
	rootCmd.Flags().BoolVarP(&cfg.StopOnCompletion, "stop-on-completion", "s", cfg.StopOnCompletion, "Exit when all todos are complete")

	// Worktree options
	rootCmd.Flags().BoolVarP(&cfg.WorktreeEnabled, "worktree", "w", cfg.WorktreeEnabled, "Run in a git worktree")
	rootCmd.Flags().StringVarP(&cfg.WorktreeBranch, "branch", "b", cfg.WorktreeBranch, "Branch name for worktree (empty = auto-generate)")
	rootCmd.Flags().BoolVarP(&cfg.WorktreeCleanup, "keep-worktree", "k", false, "Keep worktree after completion (inverts cleanup)")

	// Config file flag
	rootCmd.Flags().StringVar(&configFile, "config", "", "Path to config file (default: ./ralph.yaml or ~/.config/ralph/ralph.yaml)")

	// Handle config loading and flag processing
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Save values from explicitly set flags (they take precedence over YAML)
		savedValues := make(map[string]interface{})
		cmd.Flags().Visit(func(f *pflag.Flag) {
			switch f.Name {
			case "prompt":
				savedValues["prompt"] = cfg.PromptFile
			case "max-iterations":
				savedValues["max-iterations"] = cfg.MaxIterations
			case "max-time":
				savedValues["max-time"] = cfg.MaxTime
			case "agent-dir":
				savedValues["agent-dir"] = cfg.AgentDir
			case "cooldown":
				savedValues["cooldown"] = cfg.Cooldown
			case "verbose", "quiet":
				savedValues["verbose"] = cfg.Verbose
			case "dry-run":
				savedValues["dry-run"] = cfg.DryRun
			case "otel-enabled":
				savedValues["otel-enabled"] = cfg.OTELEnabled
			case "otel-endpoint":
				savedValues["otel-endpoint"] = cfg.OTELEndpoint
			case "metrics-prefix":
				savedValues["metrics-prefix"] = cfg.MetricsPrefix
			case "project-name":
				savedValues["project-name"] = cfg.ProjectName
			case "slack-enabled":
				savedValues["slack-enabled"] = cfg.SlackEnabled
			case "slack-webhook-url":
				savedValues["slack-webhook-url"] = cfg.SlackWebhookURL
			case "slack-channel":
				savedValues["slack-channel"] = cfg.SlackChannel
			case "slack-notify-users":
				savedValues["slack-notify-users"] = cfg.SlackNotifyUsers
			case "slack-bot-token":
				savedValues["slack-bot-token"] = cfg.SlackBotToken
			case "stop-on-completion":
				savedValues["stop-on-completion"] = cfg.StopOnCompletion
			case "worktree":
				savedValues["worktree"] = cfg.WorktreeEnabled
			case "branch":
				savedValues["branch"] = cfg.WorktreeBranch
			case "keep-worktree":
				savedValues["keep-worktree"] = true // Flag was set, invert cleanup
			}
		})

		// Load config files (global first, then local overrides)
		var configPaths []string
		if configFile != "" {
			// Explicit config file specified - use only that
			configPaths = []string{configFile}
		} else {
			// Use two-tier config: global + local
			configPaths = config.FindConfigFiles()
		}

		for _, path := range configPaths {
			if err := cfg.LoadFromFile(path); err != nil {
				return fmt.Errorf("failed to load config file %s: %w", path, err)
			}
			cfg.ConfigFile = path // Track last loaded config file
		}

		// Restore explicitly set flag values (CLI flags take precedence)
		for name, val := range savedValues {
			switch name {
			case "prompt":
				cfg.PromptFile = val.(string)
			case "max-iterations":
				cfg.MaxIterations = val.(int)
			case "max-time":
				cfg.MaxTime = val.(int)
			case "agent-dir":
				cfg.AgentDir = val.(string)
			case "cooldown":
				cfg.Cooldown = val.(int)
			case "verbose":
				cfg.Verbose = val.(bool)
			case "dry-run":
				cfg.DryRun = val.(bool)
			case "otel-enabled":
				cfg.OTELEnabled = val.(bool)
			case "otel-endpoint":
				cfg.OTELEndpoint = val.(string)
			case "metrics-prefix":
				cfg.MetricsPrefix = val.(string)
			case "project-name":
				cfg.ProjectName = val.(string)
			case "slack-enabled":
				cfg.SlackEnabled = val.(bool)
			case "slack-webhook-url":
				cfg.SlackWebhookURL = val.(string)
			case "slack-channel":
				cfg.SlackChannel = val.(string)
			case "slack-notify-users":
				cfg.SlackNotifyUsers = val.(string)
			case "slack-bot-token":
				cfg.SlackBotToken = val.(string)
			case "stop-on-completion":
				cfg.StopOnCompletion = val.(bool)
			case "worktree":
				cfg.WorktreeEnabled = val.(bool)
			case "branch":
				cfg.WorktreeBranch = val.(string)
			case "keep-worktree":
				cfg.WorktreeCleanup = false // -k inverts cleanup
			}
		}

		// Handle -q flag (inverts verbose)
		if cmd.Flags().Changed("quiet") {
			cfg.Verbose = false
		}

		return nil
	}

	// Add -v flag for version
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version")

	rootCmd.AddCommand(versionCmd)

	// Custom help template to match bash version style
	rootCmd.SetHelpTemplate(fmt.Sprintf(`ralph v%s - Run Claude in a loop

Usage: ralph [OPTIONS]

Options:
  -h, --help                  Show this help message
  -p, --prompt FILE           Path to prompt file (default: ./prompt.md)
  -n, --max-iterations N      Max loop iterations (default: 0 = unlimited)
  -t, --max-time SECONDS      Max total runtime in seconds (default: 0 = unlimited)
  -d, --agent-dir DIR         Scratchpad directory (default: ./.agent)
  -c, --cooldown SECONDS      Delay between iterations (default: 1)
  -q, --quiet                 Disable verbose output
  --dry-run                   Show what would run without executing
  --config FILE               Path to config file (default: ./ralph.yaml)

OTEL Options:
  --otel-enabled              Enable metrics export (default: false)
  --otel-endpoint URL         OTLP endpoint (default: localhost:4317)
  --metrics-prefix PREFIX     Metric name prefix (default: ralph)
  --project-name NAME         Override project label (default: cwd basename)

Slack Options:
  --slack-enabled             Enable Slack notifications (default: false)
  --slack-webhook-url URL     Slack webhook URL (or RALPH_SLACK_WEBHOOK_URL env)
  --slack-channel ID          Slack channel ID (or RALPH_SLACK_CHANNEL env)
  --slack-notify-users IDS    Comma-separated user IDs to @mention (or RALPH_SLACK_NOTIFY_USERS env)
  --slack-bot-token TOKEN     Bot token for thread replies (or RALPH_SLACK_BOT_TOKEN env)

Behavior Options:
  -s, --stop-on-completion    Exit when all todos are complete (default: false)

Worktree Options:
  -w, --worktree              Run in a git worktree (default: false)
  -b, --branch NAME           Branch name for worktree (default: auto-generate)
  -k, --keep-worktree         Keep worktree after completion (default: false)

Examples:
  ralph                           # Run forever with defaults
  ralph -n 5                      # Run for 5 iterations
  ralph -t 3600                   # Run for 1 hour
  ralph -p ~/tasks/build.md       # Use custom prompt file
  ralph -n 10 -c 5                # 10 iterations, 5s cooldown
  ralph --config ~/myconfig.yaml  # Use custom config file
  ralph -w                        # Run in a worktree (auto branch)
  ralph -w -b feature/my-task     # Run in worktree with specific branch
  ralph -w -k                     # Run in worktree, keep after completion

"I'm in danger!" - Ralph Wiggum
`, config.Version))
}

func main() {
	// Disable color output if not a terminal
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		color.NoColor = true
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
