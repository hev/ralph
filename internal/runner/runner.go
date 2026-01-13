package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fatih/color"
	"github.com/hev/ralph/internal/claude"
	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/git"
	"github.com/hev/ralph/internal/metrics"
	"github.com/hev/ralph/internal/ralph"
	"github.com/hev/ralph/internal/slack"
	"github.com/hev/ralph/internal/testmode"
	"github.com/hev/ralph/internal/todo"
	"github.com/hev/ralph/internal/worktree"
)

var (
	blue   = color.New(color.FgBlue)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
)

func log(format string, args ...interface{}) {
	blue.Print("[ralph] ")
	fmt.Printf(format+"\n", args...)
}

func logVerbose(cfg *config.Config, format string, args ...interface{}) {
	if cfg.Verbose {
		yellow.Print("[ralph] ")
		fmt.Printf(format+"\n", args...)
	}
}

func logError(format string, args ...interface{}) {
	red.Fprint(os.Stderr, "[ralph] ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func logSuccess(format string, args ...interface{}) {
	green.Print("[ralph] ")
	fmt.Printf(format+"\n", args...)
}

// Run executes the main ralph loop
func Run(cfg *config.Config) error {
	// Validate prompt file exists
	if _, err := os.Stat(cfg.PromptFile); os.IsNotExist(err) {
		logError("Prompt file not found: %s", cfg.PromptFile)
		return err
	}

	// Setup worktree if enabled
	var wtManager *worktree.Manager
	if cfg.WorktreeEnabled {
		wtManager = worktree.NewManager(cfg.WorktreeBaseDir, cfg.WorktreeBranchPrefix, cfg.WorktreeCleanup)

		// Determine branch name: explicit > issue-based > auto-generated
		branchName := cfg.WorktreeBranch
		if branchName == "" && cfg.IssueNumber > 0 {
			// Generate branch name from issue
			branchName = worktree.BranchNameFromIssue(cfg.WorktreeBranchPrefix, cfg.IssueNumber, cfg.IssueTitle)
			logVerbose(cfg, "Auto-generated branch from issue: %s", branchName)
		}

		log("Creating worktree...")
		worktreePath, err := wtManager.Create(branchName)
		if err != nil {
			logError("Failed to create worktree: %v", err)
			return err
		}
		log("Worktree created at: %s", worktreePath)
		log("Branch: %s", wtManager.GetBranchName())

		// Copy prompt file to worktree
		promptAbsPath, err := filepath.Abs(cfg.PromptFile)
		if err != nil {
			logError("Failed to get absolute path for prompt: %v", err)
			wtManager.Remove()
			return err
		}
		promptBasename := filepath.Base(cfg.PromptFile)
		worktreePromptPath := filepath.Join(worktreePath, promptBasename)
		if err := copyFile(promptAbsPath, worktreePromptPath); err != nil {
			logError("Failed to copy prompt file to worktree: %v", err)
			wtManager.Remove()
			return err
		}
		logVerbose(cfg, "Copied prompt file to worktree: %s", worktreePromptPath)

		// Change to worktree directory
		if err := os.Chdir(worktreePath); err != nil {
			logError("Failed to change to worktree directory: %v", err)
			wtManager.Remove()
			return err
		}

		// Update config paths to be relative to worktree
		cfg.PromptFile = promptBasename
		cfg.AgentDir = "./.agent"
	}

	// Create agent directory if missing
	if _, err := os.Stat(cfg.AgentDir); os.IsNotExist(err) {
		log("Creating agent directory: %s", cfg.AgentDir)
		if err := os.MkdirAll(cfg.AgentDir, 0755); err != nil {
			logError("Failed to create agent directory: %v", err)
			if wtManager != nil {
				wtManager.Remove()
			}
			return err
		}
	}

	// Load prompt
	promptBytes, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		logError("Failed to read prompt file: %v", err)
		return err
	}
	fullPrompt := string(promptBytes) + cfg.ScratchpadInstructions()

	// Show configuration
	log("Starting Ralph loop...")
	logVerbose(cfg, "Prompt file: %s", cfg.PromptFile)
	if cfg.MaxIterations > 0 {
		logVerbose(cfg, "Max iterations: %d", cfg.MaxIterations)
	} else {
		logVerbose(cfg, "Max iterations: unlimited")
	}
	if cfg.MaxTime > 0 {
		logVerbose(cfg, "Max time: %ds", cfg.MaxTime)
	} else {
		logVerbose(cfg, "Max time: unlimited")
	}
	logVerbose(cfg, "Agent dir: %s", cfg.AgentDir)
	logVerbose(cfg, "Cooldown: %ds", cfg.Cooldown)
	if cfg.Model != "" {
		logVerbose(cfg, "Model: %s", cfg.Model)
	}
	if cfg.WorktreeEnabled && wtManager != nil {
		logVerbose(cfg, "Worktree: %s", wtManager.GetWorktreePath())
		logVerbose(cfg, "Branch: %s", wtManager.GetBranchName())
	}
	if cfg.OTELEnabled {
		logVerbose(cfg, "OTEL endpoint: %s", cfg.OTELEndpoint)
		logVerbose(cfg, "Session ID: %s", cfg.SessionID)
	}

	// Dry run mode
	if cfg.DryRun {
		log("Dry run mode - would execute:")
		if cfg.Model != "" {
			fmt.Printf("claude --dangerously-skip-permissions --print --model %s -p \"$FULL_PROMPT\"\n", cfg.Model)
		} else {
			fmt.Println("claude --dangerously-skip-permissions --print -p \"$FULL_PROMPT\"")
		}
		fmt.Println()
		fmt.Println("Full prompt:")
		fmt.Println("---")
		fmt.Println(fullPrompt)
		fmt.Println("---")
		return nil
	}

	// Test mode setup
	var mockClaude *testmode.MockClaude
	if cfg.TestMode {
		log("=== TEST MODE ENABLED ===")
		log("Scenario: %s", cfg.TestScenario)
		logVerbose(cfg, "Mock Claude will simulate todo progress")

		mockClaude = testmode.NewMockClaude(cfg.TestScenario, cfg.AgentDir)

		// Set reasonable defaults for test mode
		if cfg.MaxIterations == 0 {
			cfg.MaxIterations = 3 // Default test iterations
		}
		if cfg.Cooldown == 0 {
			cfg.Cooldown = 2 // Default cooldown to prevent sound overlap
		}

		// Enable all phases for success scenario to exercise full flow
		if cfg.TestScenario == "success" {
			cfg.StopOnCompletion = true
			cfg.CodeReviewEnabled = true
			cfg.CleanupEnabled = true
			cfg.PREnabled = true
		}

		// Auto-enable sound in test mode (can still be muted with --sound-mute)
		cfg.SoundEnabled = true

		// Demonstrate all log types in test mode
		log("This is a standard log message")
		logVerbose(cfg, "This is a verbose log message")
		logError("This is an error log message (test only)")
		logSuccess("This is a success log message")
	}

	// Initialize metrics tracker
	tracker, err := metrics.NewTracker(cfg)
	if err != nil {
		logError("Failed to initialize metrics: %v", err)
		// Continue without metrics
	}

	// Initialize Slack notifier
	notifier := slack.NewNotifier(slack.NotifierConfig{
		Enabled:       cfg.SlackEnabled,
		WebhookURL:    cfg.SlackWebhookURL,
		BotToken:      cfg.SlackBotToken,
		Channel:       cfg.SlackChannel,
		NotifyUsers:   cfg.GetSlackNotifyUsers(),
		ProjectName:   cfg.ProjectName,
		SessionID:     cfg.SessionID,
		MaxIterations: cfg.MaxIterations,
		MaxTime:       cfg.MaxTime,
	})

	// Initialize sound player
	soundPlayer := ralph.NewSoundPlayer(cfg.GetSoundConfig())
	if cfg.SoundEnabled {
		logVerbose(cfg, "Ralph sounds enabled")
	}

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics session
	if tracker != nil {
		tracker.Start(ctx)
		tracker.UpdatePreviousTodos() // Initialize todo tracking
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			tracker.Stop(shutdownCtx)
		}()
	}

	// Send Slack session start notification
	if notifier.IsEnabled() {
		logVerbose(cfg, "Slack notifications enabled")
		if err := notifier.SessionStart(ctx); err != nil {
			logError("Failed to send Slack session start: %v", err)
		}
	}

	// Play session start sound
	if err := soundPlayer.PlaySessionStart(); err != nil {
		logVerbose(cfg, "Failed to play session start sound: %v", err)
	}

	// Track state
	startTime := time.Now()
	iteration := 0
	exitReason := "unknown"
	totalCommits := 0

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		exitReason = "interrupted"
		cancel()
	}()

	// Main loop
	for {
		select {
		case <-ctx.Done():
			sendSessionEnd(ctx, notifier, startTime, iteration, exitReason, totalCommits, tracker)
			printSummary(startTime, iteration, exitReason, totalCommits, tracker, "")
			cleanupWorktree(cfg, wtManager)
			return nil
		default:
		}

		iteration++

		// Check iteration limit
		if cfg.MaxIterations > 0 && iteration > cfg.MaxIterations {
			exitReason = "max iterations reached"
			break
		}

		// Check time limit
		elapsed := time.Since(startTime)
		if cfg.MaxTime > 0 && int(elapsed.Seconds()) >= cfg.MaxTime {
			exitReason = "max time reached"
			break
		}

		log("=== Iteration %d ===", iteration)
		log("Running claude (this may take a moment)...")

		// Track iteration timing
		if tracker != nil {
			tracker.BeforeIteration()
		}
		iterationStart := time.Now()

		// Run claude with streaming output (or mock in test mode)
		var exitCode int
		if mockClaude != nil {
			exitCode, err = mockClaude.RunIteration(ctx)
			// Stream mock output
			for line := range mockClaude.StreamOutput() {
				claude.ParseAndPrint(line)
			}
		} else {
			exitCode, err = runClaude(ctx, fullPrompt, cfg.Model)
		}
		iterationDuration := time.Since(iterationStart)
		hadError := err != nil

		if hadError {
			if ctx.Err() != nil {
				// Context was cancelled (signal received)
				break
			}
			logError("Claude exited with error (code %d), continuing to next iteration...", exitCode)
			if tracker != nil {
				tracker.RecordError(ctx, "execution_error")
			}
		} else {
			logSuccess("Iteration %d complete", iteration)
		}

		// Play Ralph Wiggum quote after each iteration
		if err := soundPlayer.Play(); err != nil {
			logVerbose(cfg, "Failed to play sound: %v", err)
		}

		// Record iteration metrics
		if tracker != nil {
			tracker.AfterIteration(ctx, iterationDuration, hadError, "complete")
			totalCommits = tracker.GetCommitsDelta()

			// Check for newly started todos and send Slack notifications
			if notifier.IsEnabled() {
				newlyStarted := tracker.GetNewlyInProgressTodos()
				counts, _ := tracker.GetTodoCounts()
				for _, itemWithIdx := range newlyStarted {
					if err := notifier.TodoStarted(ctx, itemWithIdx.Item.Text, itemWithIdx.Index, counts.Total(), iteration, counts.Completed); err != nil {
						logError("Failed to send Slack todo started notification: %v", err)
					}
				}
			}

			// Check for newly completed todos and send Slack notifications
			if notifier.IsEnabled() {
				newlyCompleted := tracker.GetNewlyCompletedTodos()
				counts, _ := tracker.GetTodoCounts()
				for _, item := range newlyCompleted {
					if err := notifier.TodoCompleted(ctx, item.Text, counts.Completed, counts.Total(), iteration, totalCommits, iterationDuration); err != nil {
						logError("Failed to send Slack todo notification: %v", err)
					}
				}
			}

			// Update previous todos for next iteration comparison
			tracker.UpdatePreviousTodos()
		}

		// Check for stop-on-completion (works even without tracker)
		if cfg.StopOnCompletion {
			todoPath := filepath.Join(cfg.AgentDir, "TODO.md")
			if counts, err := todo.ParseFile(todoPath); err == nil {
				if counts.Pending == 0 && counts.Completed > 0 {
					exitReason = "all todos complete"
					log("All todos complete, stopping...")
					// Play todo list complete sound
					if err := soundPlayer.PlayTodoComplete(); err != nil {
						logVerbose(cfg, "Failed to play todo complete sound: %v", err)
					}
					break
				}
			}
		}

		logVerbose(cfg, "Sleeping for %ds...", cfg.Cooldown)

		// Sleep with context awareness
		select {
		case <-ctx.Done():
			sendSessionEnd(ctx, notifier, startTime, iteration, exitReason, totalCommits, tracker)
			printSummary(startTime, iteration, exitReason, totalCommits, tracker, "")
			cleanupWorktree(cfg, wtManager)
			return nil
		case <-time.After(time.Duration(cfg.Cooldown) * time.Second):
		}
	}

	// Run code review phase if enabled and todos completed
	if cfg.CodeReviewEnabled && exitReason == "all todos complete" {
		reviewExitReason, reviewIters := runCodeReviewPhase(ctx, cfg, notifier, tracker, soundPlayer, mockClaude)
		exitReason = reviewExitReason
		iteration += reviewIters
	}

	// Run cleanup phase if enabled
	if cfg.CleanupEnabled {
		cleanupExitReason := runCleanupPhase(ctx, cfg, notifier)
		if cleanupExitReason != "" {
			exitReason = cleanupExitReason
		}
	}

	// Run PR creation phase if enabled
	var prURL string
	if cfg.PREnabled {
		var prErr error
		prURL, prErr = runPRPhase(ctx, cfg, notifier, tracker)
		if prErr != nil {
			logError("PR creation failed: %v", prErr)
		}
	}

	sendSessionEnd(ctx, notifier, startTime, iteration-1, exitReason, totalCommits, tracker)
	printSummary(startTime, iteration-1, exitReason, totalCommits, tracker, prURL)
	cleanupWorktree(cfg, wtManager)
	return nil
}

// runCodeReviewPhase runs the code review loop after todos are complete
// Returns the exit reason and number of iterations
func runCodeReviewPhase(ctx context.Context, cfg *config.Config, notifier *slack.Notifier, tracker *metrics.Tracker, soundPlayer *ralph.SoundPlayer, mockClaude *testmode.MockClaude) (string, int) {
	log("=== Starting Code Review Phase ===")

	// Set mock to code review phase if in test mode
	if mockClaude != nil {
		mockClaude.SetPhase("code_review")
	}

	// Get the code review prompt
	reviewPrompt := cfg.CodeReviewInstructions()

	// Clear the TODO file to prepare for review issues (skip in test mode, mock will write it)
	todoPath := filepath.Join(cfg.AgentDir, "TODO.md")
	if mockClaude == nil {
		if err := os.WriteFile(todoPath, []byte("# Code Review\n\n## Issues Found\n\n"), 0644); err != nil {
			logError("Failed to clear TODO file for code review: %v", err)
			return "code review setup failed", 0
		}
	}

	// Reset todo tracking for review phase
	if tracker != nil {
		tracker.UpdatePreviousTodos()
	}

	reviewStartTime := time.Now()
	reviewIteration := 0
	issuesFound := 0
	issuesFixed := 0

	for cfg.CodeReviewMaxIterations == 0 || reviewIteration < cfg.CodeReviewMaxIterations {
		select {
		case <-ctx.Done():
			return "interrupted during code review", reviewIteration
		default:
		}

		reviewIteration++
		if cfg.CodeReviewMaxIterations == 0 {
			log("=== Code Review Iteration %d (unlimited) ===", reviewIteration)
		} else {
			log("=== Code Review Iteration %d of %d ===", reviewIteration, cfg.CodeReviewMaxIterations)
		}

		// Send Slack notification for review iteration
		if notifier.IsEnabled() {
			if err := notifier.CodeReviewStarted(ctx, reviewIteration, cfg.CodeReviewMaxIterations); err != nil {
				logError("Failed to send code review started notification: %v", err)
			}
		}

		// Track iteration timing
		if tracker != nil {
			tracker.BeforeIteration()
		}
		iterationStart := time.Now()

		// Run claude with review prompt (or mock in test mode)
		// Use code review model if specified, otherwise fall back to main model
		model := cfg.CodeReviewModel
		if model == "" {
			model = cfg.Model
		}
		var exitCode int
		var err error
		if mockClaude != nil {
			exitCode, err = mockClaude.RunIteration(ctx)
			// Stream mock output
			for line := range mockClaude.StreamOutput() {
				claude.ParseAndPrint(line)
			}
		} else {
			exitCode, err = runClaude(ctx, reviewPrompt, model)
		}
		iterationDuration := time.Since(iterationStart)
		hadError := err != nil

		if hadError {
			if ctx.Err() != nil {
				return "interrupted during code review", reviewIteration
			}
			logError("Claude exited with error (code %d) during code review", exitCode)
			if tracker != nil {
				tracker.RecordError(ctx, "code_review_error")
			}
		} else {
			logSuccess("Code review iteration %d complete", reviewIteration)
		}

		// Play Ralph Wiggum quote after each code review iteration
		if soundPlayer != nil {
			if err := soundPlayer.Play(); err != nil {
				logVerbose(cfg, "Failed to play sound: %v", err)
			}
		}

		// Record iteration metrics
		if tracker != nil {
			tracker.AfterIteration(ctx, iterationDuration, hadError, "code_review")

			// Get current todo counts
			counts, err := tracker.GetTodoCounts()
			if err == nil {
				// Track issues found (total todos created during review)
				if reviewIteration == 1 {
					issuesFound = counts.Total()
				}
				issuesFixed = counts.Completed

				// Check if all review issues are resolved
				if counts.Pending == 0 && counts.Completed > 0 {
					log("All code review issues resolved")
					if notifier.IsEnabled() {
						reviewDuration := time.Since(reviewStartTime)
						if err := notifier.CodeReviewComplete(ctx, reviewIteration, issuesFound, issuesFixed, reviewDuration); err != nil {
							logError("Failed to send code review complete notification: %v", err)
						}
					}
					return "code review complete", reviewIteration
				}

				// Check if review found no issues (indicated by a single completed item)
				if counts.Pending == 0 && counts.Total() == 0 {
					log("Code review found no issues")
					if notifier.IsEnabled() {
						reviewDuration := time.Since(reviewStartTime)
						if err := notifier.CodeReviewComplete(ctx, reviewIteration, 0, 0, reviewDuration); err != nil {
							logError("Failed to send code review complete notification: %v", err)
						}
					}
					return "code review complete - no issues", reviewIteration
				}
			}

			// Update previous todos for next iteration
			tracker.UpdatePreviousTodos()
		}

		// Sleep between iterations
		if cfg.CodeReviewMaxIterations == 0 || reviewIteration < cfg.CodeReviewMaxIterations {
			logVerbose(cfg, "Sleeping for %ds...", cfg.Cooldown)
			select {
			case <-ctx.Done():
				return "interrupted during code review", reviewIteration
			case <-time.After(time.Duration(cfg.Cooldown) * time.Second):
			}
		}
	}

	// Only reached when max iterations is hit (not in unlimited mode)
	if cfg.CodeReviewMaxIterations > 0 {
		log("Code review max iterations reached")
		if notifier.IsEnabled() {
			reviewDuration := time.Since(reviewStartTime)
			if err := notifier.CodeReviewComplete(ctx, reviewIteration, issuesFound, issuesFixed, reviewDuration); err != nil {
				logError("Failed to send code review complete notification: %v", err)
			}
		}
	}

	return "code review max iterations reached", reviewIteration
}

// runCleanupPhase removes artifacts based on configured patterns
// Returns an exit reason if cleanup changes the session outcome
func runCleanupPhase(ctx context.Context, cfg *config.Config, notifier *slack.Notifier) string {
	log("=== Starting Cleanup Phase ===")

	cleanupStartTime := time.Now()

	// Send Slack notification
	if notifier.IsEnabled() {
		if err := notifier.CleanupStarted(ctx, len(cfg.CleanupPatterns)); err != nil {
			logError("Failed to send cleanup started notification: %v", err)
		}
	}

	// Find and remove files matching patterns
	filesRemoved := 0
	for _, pattern := range cfg.CleanupPatterns {
		select {
		case <-ctx.Done():
			return "interrupted during cleanup"
		default:
		}

		logVerbose(cfg, "Scanning pattern: %s", pattern)

		// Use doublestar for glob matching
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			logError("Failed to glob pattern %s: %v", pattern, err)
			continue
		}

		for _, match := range matches {
			// Skip directories
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				continue
			}

			log("Removing: %s", match)
			if err := os.Remove(match); err != nil {
				logError("Failed to remove %s: %v", match, err)
			} else {
				filesRemoved++
			}
		}
	}

	cleanupDuration := time.Since(cleanupStartTime)

	// Send completion notification
	if notifier.IsEnabled() {
		if err := notifier.CleanupComplete(ctx, filesRemoved, cleanupDuration); err != nil {
			logError("Failed to send cleanup complete notification: %v", err)
		}
	}

	if filesRemoved > 0 {
		logSuccess("Cleanup complete: removed %d file(s)", filesRemoved)
	} else {
		logSuccess("Cleanup complete: no artifacts found")
	}

	return ""
}

// runPRPhase creates a pull request for the changes made
// Returns the PR URL if successful, or an error
func runPRPhase(ctx context.Context, cfg *config.Config, notifier *slack.Notifier, tracker *metrics.Tracker) (string, error) {
	log("=== Starting PR Creation Phase ===")

	// Test mode: simulate PR creation
	if cfg.TestMode {
		log("Test mode: Simulating PR creation...")
		prURL := "https://github.com/test/repo/pull/999"
		prTitle := "Test PR Title"
		logSuccess("PR created (simulated): %s", prURL)

		// Send Slack notification with simulated URL
		if notifier.IsEnabled() {
			if err := notifier.PRCreated(ctx, prURL, prTitle); err != nil {
				logError("Failed to send Slack PR notification: %v", err)
			}
		}

		return prURL, nil
	}

	// Check if we're on a branch other than default
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	defaultBranch, err := git.GetDefaultBranch()
	if err != nil {
		defaultBranch = "main"
	}

	if currentBranch == defaultBranch {
		logError("Cannot create PR: currently on default branch (%s)", defaultBranch)
		return "", fmt.Errorf("cannot create PR from default branch")
	}

	// Push the branch if not already pushed
	if !git.IsBranchPushed() {
		log("Pushing branch %s to remote...", currentBranch)
		if err := git.PushBranch(); err != nil {
			return "", fmt.Errorf("failed to push branch: %w", err)
		}
	}

	// Generate PR body
	baseBranch := cfg.PRBase
	if baseBranch == "" {
		baseBranch = defaultBranch
	}

	body := generatePRBody(cfg, tracker, baseBranch)

	// Generate title if not provided
	title := cfg.PRTitle
	if title == "" {
		title = generatePRTitle(currentBranch)
	}

	// Create the PR
	log("Creating pull request...")
	prConfig := git.PRConfig{
		Title: title,
		Base:  baseBranch,
		Body:  body,
	}

	result, err := git.CreatePR(prConfig)
	if err != nil {
		return "", err
	}

	logSuccess("PR created: %s", result.URL)

	// Send Slack notification
	if notifier.IsEnabled() {
		if err := notifier.PRCreated(ctx, result.URL, title); err != nil {
			logError("Failed to send Slack PR notification: %v", err)
		}
	}

	return result.URL, nil
}

// generatePRTitle generates a PR title from the branch name
func generatePRTitle(branchName string) string {
	// Remove common prefixes
	title := branchName
	for _, prefix := range []string{"ralph/", "feature/", "fix/", "bugfix/", "hotfix/"} {
		title = strings.TrimPrefix(title, prefix)
	}

	// Convert dashes/underscores to spaces and capitalize
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitalize first letter
	if len(title) > 0 {
		title = strings.ToUpper(string(title[0])) + title[1:]
	}

	return title
}

// generatePRBody generates a PR body with summary of changes
func generatePRBody(cfg *config.Config, tracker *metrics.Tracker, baseBranch string) string {
	var body strings.Builder

	body.WriteString("## Summary\n\n")

	// Add issue reference if this PR was created from an issue
	if cfg.IssueNumber > 0 {
		body.WriteString(fmt.Sprintf("Fixes #%d\n\n", cfg.IssueNumber))
	}

	body.WriteString("Changes made by Ralph automated loop.\n\n")

	// Add todo summary if available
	if tracker != nil {
		if counts, err := tracker.GetTodoCounts(); err == nil && counts.Total() > 0 {
			body.WriteString(fmt.Sprintf("- Completed %d/%d tasks (%.0f%%)\n", counts.Completed, counts.Total(), counts.CompletionRate()))
		}
	}

	// Add commit summary
	commits, err := git.GetCommitsSinceBase(baseBranch)
	if err == nil && len(commits) > 0 {
		body.WriteString(fmt.Sprintf("- Made %d commits\n", len(commits)))

		if len(commits) <= 10 {
			body.WriteString("\n### Commits\n\n")
			for _, commit := range commits {
				body.WriteString(fmt.Sprintf("- %s\n", commit))
			}
		}
	}

	// Add source issue link if available
	if cfg.IssueURL != "" {
		body.WriteString(fmt.Sprintf("\n## Source Issue\n\n[Issue #%d](%s)\n", cfg.IssueNumber, cfg.IssueURL))
	}

	body.WriteString("\n---\n\n")
	body.WriteString("*Generated by [Ralph](https://github.com/hev/ralph)*\n")

	return body.String()
}

func sendSessionEnd(ctx context.Context, notifier *slack.Notifier, startTime time.Time, iterations int, exitReason string, commits int, tracker *metrics.Tracker) {
	if !notifier.IsEnabled() {
		return
	}

	elapsed := time.Since(startTime)
	summary := slack.SessionSummary{
		Iterations: iterations,
		Duration:   elapsed,
		Commits:    commits,
		ExitReason: exitReason,
	}

	if tracker != nil {
		if counts, err := tracker.GetTodoCounts(); err == nil {
			summary.TodosDone = counts.Completed
			summary.TodosTotal = counts.Total()
		}
	}

	if err := notifier.SessionEnd(ctx, summary); err != nil {
		logError("Failed to send Slack session end: %v", err)
	}
}

func runClaude(ctx context.Context, prompt string, model string) (int, error) {
	client, err := claude.NewClient(ctx, prompt, model)
	if err != nil {
		return -1, err
	}

	if err := client.Start(); err != nil {
		return -1, err
	}

	// Stream and parse output
	lines := client.StreamOutput()
	for line := range lines {
		claude.ParseAndPrint(line)
	}

	return client.Wait()
}

func printSummary(startTime time.Time, iterations int, exitReason string, commits int, tracker *metrics.Tracker, prURL string) {
	elapsed := time.Since(startTime)
	fmt.Println()
	logSuccess("=== Ralph Summary ===")
	logSuccess("Iterations completed: %d", iterations)
	logSuccess("Total time: %.0fs", elapsed.Seconds())
	logSuccess("Exit reason: %s", exitReason)
	logSuccess("Commits made: %d", commits)

	// Show todo status if available
	if tracker != nil {
		if counts, err := tracker.GetTodoCounts(); err == nil && counts.Total() > 0 {
			logSuccess("Todos: %d/%d complete (%.0f%%)", counts.Completed, counts.Total(), counts.CompletionRate())
		}
	}

	// Show PR URL if created
	if prURL != "" {
		logSuccess("PR created: %s", prURL)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// cleanupWorktree pushes the branch and removes the worktree if cleanup is enabled
func cleanupWorktree(cfg *config.Config, wtManager *worktree.Manager) {
	if wtManager == nil {
		return
	}

	// Push branch to remote before cleanup
	log("Pushing branch %s to remote...", wtManager.GetBranchName())
	if err := wtManager.Push(); err != nil {
		logError("Failed to push branch: %v", err)
		// Continue with cleanup even if push fails
	} else {
		logSuccess("Branch pushed successfully")
	}

	// Cleanup worktree if enabled
	if cfg.WorktreeCleanup {
		log("Cleaning up worktree...")
		if err := wtManager.Remove(); err != nil {
			logError("Failed to remove worktree: %v", err)
		} else {
			logSuccess("Worktree removed")
		}
	} else {
		log("Keeping worktree at: %s", wtManager.GetWorktreePath())
	}
}
