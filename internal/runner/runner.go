package runner

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/hev/ralph/internal/claude"
	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/metrics"
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

	// Create agent directory if missing
	if _, err := os.Stat(cfg.AgentDir); os.IsNotExist(err) {
		log("Creating agent directory: %s", cfg.AgentDir)
		if err := os.MkdirAll(cfg.AgentDir, 0755); err != nil {
			logError("Failed to create agent directory: %v", err)
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

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics session
	if tracker != nil {
		tracker.Start(ctx)
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			tracker.Stop(shutdownCtx)
		}()
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
			printSummary(startTime, iteration, exitReason, totalCommits, tracker)
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
		}

		logVerbose(cfg, "Sleeping for %ds...", cfg.Cooldown)

		// Sleep with context awareness
		select {
		case <-ctx.Done():
			printSummary(startTime, iteration, exitReason, totalCommits, tracker)
			return nil
		case <-time.After(time.Duration(cfg.Cooldown) * time.Second):
		}
	}

	printSummary(startTime, iteration-1, exitReason, totalCommits, tracker)
	return nil
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
