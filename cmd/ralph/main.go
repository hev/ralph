package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/github"
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
	rootCmd.Flags().StringVar(&cfg.Model, "model", cfg.Model, "Model to use (e.g., sonnet, opus, haiku)")

	// Behavior options
	rootCmd.Flags().BoolVarP(&cfg.StopOnCompletion, "stop-on-completion", "s", cfg.StopOnCompletion, "Exit when all todos are complete")

	// Code review options
	rootCmd.Flags().BoolVar(&cfg.CodeReviewEnabled, "code-review", cfg.CodeReviewEnabled, "Run code review phase after todos complete")
	rootCmd.Flags().IntVar(&cfg.CodeReviewMaxIterations, "code-review-max-iterations", cfg.CodeReviewMaxIterations, "Max iterations for code review phase")
	rootCmd.Flags().StringVar(&cfg.CodeReviewPrompt, "code-review-prompt", cfg.CodeReviewPrompt, "Custom prompt for code review phase")
	rootCmd.Flags().StringVar(&cfg.CodeReviewModel, "code-review-model", cfg.CodeReviewModel, "Model for code review phase (defaults to --model)")

	// Cleanup options
	rootCmd.Flags().BoolVar(&cfg.CleanupEnabled, "cleanup", cfg.CleanupEnabled, "Run cleanup phase after code review")
	rootCmd.Flags().StringVar(&cfg.CleanupModel, "cleanup-model", cfg.CleanupModel, "Model for cleanup phase (defaults to --model)")

	// PR options
	rootCmd.Flags().BoolVar(&cfg.PREnabled, "pr", cfg.PREnabled, "Create a PR when the loop completes")
	rootCmd.Flags().StringVar(&cfg.PRTitle, "pr-title", cfg.PRTitle, "Custom title for the PR (empty = auto-generate)")
	rootCmd.Flags().StringVar(&cfg.PRBase, "pr-base", cfg.PRBase, "Base branch for the PR (empty = default branch)")

	// GitHub issue options
	rootCmd.Flags().StringVarP(&cfg.Issue, "issue", "i", cfg.Issue, "GitHub issue number or URL (generates prompt from issue)")

	// Worktree options
	rootCmd.Flags().BoolVarP(&cfg.WorktreeEnabled, "worktree", "w", cfg.WorktreeEnabled, "Run in a git worktree")
	rootCmd.Flags().StringVarP(&cfg.WorktreeBranch, "branch", "b", cfg.WorktreeBranch, "Branch name for worktree (empty = auto-generate)")
	rootCmd.Flags().BoolVarP(&cfg.WorktreeCleanup, "keep-worktree", "k", false, "Keep worktree after completion (inverts cleanup)")

	// Sound options
	rootCmd.Flags().BoolVar(&cfg.SoundEnabled, "sound", cfg.SoundEnabled, "Play Ralph Wiggum quotes after each iteration")
	rootCmd.Flags().BoolVar(&cfg.SoundMute, "sound-mute", cfg.SoundMute, "Mute sound playback (or RALPH_SOUND_MUTE=1 env)")

	// Test mode options
	rootCmd.Flags().BoolVar(&cfg.TestMode, "test-mode", cfg.TestMode, "Run in test mode (mock Claude, simulate todo progress)")
	rootCmd.Flags().StringVar(&cfg.TestScenario, "test-scenario", cfg.TestScenario, "Test scenario: success, error, partial")

	// State logging options
	rootCmd.Flags().BoolVar(&cfg.StateLoggingEnabled, "state-logging", cfg.StateLoggingEnabled, "Enable automatic state logging")
	rootCmd.Flags().IntVar(&cfg.RunsRetention, "runs-retention", cfg.RunsRetention, "Number of run logs to keep in runs/")

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
			case "model":
				savedValues["model"] = cfg.Model
			case "stop-on-completion":
				savedValues["stop-on-completion"] = cfg.StopOnCompletion
			case "code-review":
				savedValues["code-review"] = cfg.CodeReviewEnabled
			case "code-review-max-iterations":
				savedValues["code-review-max-iterations"] = cfg.CodeReviewMaxIterations
			case "code-review-prompt":
				savedValues["code-review-prompt"] = cfg.CodeReviewPrompt
			case "code-review-model":
				savedValues["code-review-model"] = cfg.CodeReviewModel
			case "cleanup":
				savedValues["cleanup"] = cfg.CleanupEnabled
			case "cleanup-model":
				savedValues["cleanup-model"] = cfg.CleanupModel
			case "pr":
				savedValues["pr"] = cfg.PREnabled
			case "pr-title":
				savedValues["pr-title"] = cfg.PRTitle
			case "pr-base":
				savedValues["pr-base"] = cfg.PRBase
			case "issue":
				savedValues["issue"] = cfg.Issue
			case "worktree":
				savedValues["worktree"] = cfg.WorktreeEnabled
			case "branch":
				savedValues["branch"] = cfg.WorktreeBranch
			case "keep-worktree":
				savedValues["keep-worktree"] = true // Flag was set, invert cleanup
			case "sound":
				savedValues["sound"] = cfg.SoundEnabled
			case "sound-mute":
				savedValues["sound-mute"] = cfg.SoundMute
			case "test-mode":
				savedValues["test-mode"] = cfg.TestMode
			case "test-scenario":
				savedValues["test-scenario"] = cfg.TestScenario
			case "state-logging":
				savedValues["state-logging"] = cfg.StateLoggingEnabled
			case "runs-retention":
				savedValues["runs-retention"] = cfg.RunsRetention
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
			case "model":
				cfg.Model = val.(string)
			case "stop-on-completion":
				cfg.StopOnCompletion = val.(bool)
			case "code-review":
				cfg.CodeReviewEnabled = val.(bool)
			case "code-review-max-iterations":
				cfg.CodeReviewMaxIterations = val.(int)
			case "code-review-prompt":
				cfg.CodeReviewPrompt = val.(string)
			case "code-review-model":
				cfg.CodeReviewModel = val.(string)
			case "cleanup":
				cfg.CleanupEnabled = val.(bool)
			case "cleanup-model":
				cfg.CleanupModel = val.(string)
			case "pr":
				cfg.PREnabled = val.(bool)
			case "pr-title":
				cfg.PRTitle = val.(string)
			case "pr-base":
				cfg.PRBase = val.(string)
			case "issue":
				cfg.Issue = val.(string)
			case "worktree":
				cfg.WorktreeEnabled = val.(bool)
			case "branch":
				cfg.WorktreeBranch = val.(string)
			case "keep-worktree":
				cfg.WorktreeCleanup = false // -k inverts cleanup
			case "sound":
				cfg.SoundEnabled = val.(bool)
			case "sound-mute":
				cfg.SoundMute = val.(bool)
			case "test-mode":
				cfg.TestMode = val.(bool)
			case "test-scenario":
				cfg.TestScenario = val.(string)
			case "state-logging":
				cfg.StateLoggingEnabled = val.(bool)
			case "runs-retention":
				cfg.RunsRetention = val.(int)
			}
		}

		// Apply stop-on-completion default for code review max iterations
		// Priority: explicit flag > stop-on-completion default > config default (0/unlimited)
		if _, explicitlySet := savedValues["code-review-max-iterations"]; !explicitlySet {
			if cfg.StopOnCompletion && cfg.CodeReviewMaxIterations == 0 {
				cfg.CodeReviewMaxIterations = 1
			}
		}

		// Handle -q flag (inverts verbose)
		if cmd.Flags().Changed("quiet") {
			cfg.Verbose = false
		}

		// Backward compatibility: fall back to ./prompt.md if new default doesn't exist
		if !cmd.Flags().Changed("prompt") {
			if _, err := os.Stat(cfg.PromptFile); os.IsNotExist(err) {
				// New default (.agent/IMPLEMENTATION_PLAN.md) doesn't exist
				oldDefault := "./prompt.md"
				if _, err := os.Stat(oldDefault); err == nil {
					// Old default exists, use it for backward compatibility
					cfg.PromptFile = oldDefault
				}
			}
		}

		// Handle --issue flag: fetch issue and generate prompt
		if cfg.Issue != "" {
			// Check if -p was also provided (conflict)
			if cmd.Flags().Changed("prompt") {
				return fmt.Errorf("cannot use both --issue and --prompt flags")
			}

			// Fetch the issue
			issue, err := github.FetchIssueFromRef(cfg.Issue)
			if err != nil {
				return fmt.Errorf("failed to fetch issue: %w", err)
			}

			// Store the issue info for later use (e.g., PR body, branch name)
			cfg.IssueNumber = issue.Number
			cfg.IssueTitle = issue.Title
			cfg.IssueURL = issue.URL

			// Generate prompt content
			promptContent := github.GeneratePrompt(issue)

			// Write to prompt.md
			promptPath := filepath.Join(".", "prompt.md")
			if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
				return fmt.Errorf("failed to write prompt file: %w", err)
			}

			// Update config to use the generated prompt
			cfg.PromptFile = promptPath

			// Log that we generated the prompt from an issue
			fmt.Printf("[ralph] Generated prompt from issue #%d: %s\n", issue.Number, issue.Title)
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
  -p, --prompt FILE           Path to prompt file (default: .agent/IMPLEMENTATION_PLAN.md)
  -n, --max-iterations N      Max loop iterations (default: 0 = unlimited)
  -t, --max-time SECONDS      Max total runtime in seconds (default: 0 = unlimited)
  -d, --agent-dir DIR         Scratchpad directory (default: ./.agent)
  -c, --cooldown SECONDS      Delay between iterations (default: 1)
  -q, --quiet                 Disable verbose output
  --dry-run                   Show what would run without executing
  --config FILE               Path to config file (default: ./ralph.yaml)
  --model MODEL               Model to use (e.g., sonnet, opus, haiku)

Behavior Options:
  -s, --stop-on-completion    Exit when all todos are complete (default: false)

Code Review Options:
  --code-review               Run code review phase after todos complete (default: false)
  --code-review-max-iterations N  Max iterations for code review phase (default: 0=unlimited, 1 with -s)
  --code-review-prompt TEXT   Custom prompt for code review phase
  --code-review-model MODEL   Model for code review phase (defaults to --model)

Cleanup Options:
  --cleanup                   Run cleanup phase after code review (default: false)
  --cleanup-model MODEL       Model for cleanup phase (defaults to --model)

PR Options:
  --pr                        Create a PR when the loop completes (default: false)
  --pr-title TITLE            Custom title for the PR (default: auto-generate)
  --pr-base BRANCH            Base branch for the PR (default: repo default)

GitHub Issue Options:
  -i, --issue REF             GitHub issue number or URL (generates prompt from issue)
                              Formats: 42, #42, owner/repo#42, or full URL

Worktree Options:
  -w, --worktree              Run in a git worktree (default: false)
  -b, --branch NAME           Branch name for worktree (default: auto-generate)
  -k, --keep-worktree         Keep worktree after completion (default: false)

Sound Options:
  --sound                     Play Ralph Wiggum quotes after each iteration (default: false)
  --sound-mute                Mute sound playback (or RALPH_SOUND_MUTE=1 env)

Test Mode Options:
  --test-mode                 Run in test mode (mock Claude, simulate todo progress)
  --test-scenario SCENARIO    Test scenario: success, error, partial (default: success)

State Logging Options:
  --state-logging             Enable automatic state logging (default: true)
  --runs-retention N          Number of run logs to keep in runs/ (default: 50)

Examples:
  ralph                           # Run forever with defaults
  ralph -n 5                      # Run for 5 iterations
  ralph -t 3600                   # Run for 1 hour
  ralph -p ~/tasks/build.md       # Use custom prompt file
  ralph -n 10 -c 5                # 10 iterations, 5s cooldown
  ralph --config ~/myconfig.yaml  # Use custom config file
  ralph --model opus              # Use opus model
  ralph --model sonnet --code-review-model opus  # Different models per phase
  ralph -w                        # Run in a worktree (auto branch)
  ralph -w -b feature/my-task     # Run in worktree with specific branch
  ralph -w -k                     # Run in worktree, keep after completion
  ralph -s --code-review          # Stop on completion, then run code review
  ralph -s --code-review --cleanup # Full pipeline: work, review, cleanup
  ralph -s --pr                    # Stop on completion, create PR
  ralph -w -s --pr                 # Worktree + stop + PR (common pattern)
  ralph -i 42                      # Start loop from issue #42
  ralph -i owner/repo#42           # Issue from specific repo
  ralph -i 42 -w                   # Issue + worktree (auto branch from issue)
  ralph -i 42 -w --pr              # Full workflow: issue -> worktree -> PR
  ralph --sound                     # Play Ralph Wiggum quotes after each iteration
  ralph --test-mode                  # Run in test mode (mock Claude)

OTEL and Slack notifications are configured via config file or environment variables.
See: https://github.com/hev/ralph#configuration

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
