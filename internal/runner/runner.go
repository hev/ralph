package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/hev/ralph/internal/claude"
	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/metrics"
	"github.com/hev/ralph/internal/slack"
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

		log("Creating worktree...")
		worktreePath, err := wtManager.Create(cfg.WorktreeBranch)
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
		fmt.Println("claude --dangerously-skip-permissions --print -p \"$FULL_PROMPT\"")
		fmt.Println()
		fmt.Println("Full prompt:")
		fmt.Println("---")
		fmt.Println(fullPrompt)
		fmt.Println("---")
		return nil
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
			printSummary(startTime, iteration, exitReason, totalCommits, tracker)
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

		// Run claude with streaming output
		exitCode, err := runClaude(ctx, fullPrompt)
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

			// Check for stop-on-completion
			if cfg.StopOnCompletion {
				if counts, err := tracker.GetTodoCounts(); err == nil {
					if counts.Pending == 0 && counts.Completed > 0 {
						exitReason = "all todos complete"
						log("All todos complete, stopping...")
						break
					}
				}
			}
		}

		logVerbose(cfg, "Sleeping for %ds...", cfg.Cooldown)

		// Sleep with context awareness
		select {
		case <-ctx.Done():
			sendSessionEnd(ctx, notifier, startTime, iteration, exitReason, totalCommits, tracker)
			printSummary(startTime, iteration, exitReason, totalCommits, tracker)
			cleanupWorktree(cfg, wtManager)
			return nil
		case <-time.After(time.Duration(cfg.Cooldown) * time.Second):
		}
	}

	// Run code review phase if enabled and todos completed
	if cfg.CodeReviewEnabled && exitReason == "all todos complete" {
		reviewExitReason, reviewIters := runCodeReviewPhase(ctx, cfg, notifier, tracker)
		exitReason = reviewExitReason
		iteration += reviewIters
	}

	sendSessionEnd(ctx, notifier, startTime, iteration-1, exitReason, totalCommits, tracker)
	printSummary(startTime, iteration-1, exitReason, totalCommits, tracker)
	cleanupWorktree(cfg, wtManager)
	return nil
}

// runCodeReviewPhase runs the code review loop after todos are complete
// Returns the exit reason and number of iterations
func runCodeReviewPhase(ctx context.Context, cfg *config.Config, notifier *slack.Notifier, tracker *metrics.Tracker) (string, int) {
	log("=== Starting Code Review Phase ===")

	// Get the code review prompt
	reviewPrompt := cfg.CodeReviewInstructions()

	// Clear the TODO file to prepare for review issues
	todoPath := filepath.Join(cfg.AgentDir, "TODO.md")
	if err := os.WriteFile(todoPath, []byte("# Code Review\n\n## Issues Found\n\n"), 0644); err != nil {
		logError("Failed to clear TODO file for code review: %v", err)
		return "code review setup failed", 0
	}

	// Reset todo tracking for review phase
	if tracker != nil {
		tracker.UpdatePreviousTodos()
	}

	reviewStartTime := time.Now()
	reviewIteration := 0
	issuesFound := 0
	issuesFixed := 0

	for reviewIteration < cfg.CodeReviewMaxIterations {
		select {
		case <-ctx.Done():
			return "interrupted during code review", reviewIteration
		default:
		}

		reviewIteration++
		log("=== Code Review Iteration %d of %d ===", reviewIteration, cfg.CodeReviewMaxIterations)

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

		// Run claude with review prompt
		exitCode, err := runClaude(ctx, reviewPrompt)
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
		if reviewIteration < cfg.CodeReviewMaxIterations {
			logVerbose(cfg, "Sleeping for %ds...", cfg.Cooldown)
			select {
			case <-ctx.Done():
				return "interrupted during code review", reviewIteration
			case <-time.After(time.Duration(cfg.Cooldown) * time.Second):
			}
		}
	}

	log("Code review max iterations reached")
	if notifier.IsEnabled() {
		reviewDuration := time.Since(reviewStartTime)
		if err := notifier.CodeReviewComplete(ctx, reviewIteration, issuesFound, issuesFixed, reviewDuration); err != nil {
			logError("Failed to send code review complete notification: %v", err)
		}
	}

	return "code review max iterations reached", reviewIteration
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

func runClaude(ctx context.Context, prompt string) (int, error) {
	client, err := claude.NewClient(ctx, prompt)
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

func printSummary(startTime time.Time, iterations int, exitReason string, commits int, tracker *metrics.Tracker) {
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
