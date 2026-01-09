package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// PRConfig holds configuration for creating a pull request
type PRConfig struct {
	Title string // Custom title (empty = auto-generate)
	Base  string // Base branch (empty = repo default)
	Body  string // PR body/description
}

// PRResult contains the result of PR creation
type PRResult struct {
	URL    string // The URL of the created PR
	Number int    // The PR number
}

// CreatePR creates a pull request using the gh CLI
func CreatePR(cfg PRConfig) (*PRResult, error) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found: please install GitHub CLI (https://cli.github.com)")
	}

	// Build the gh pr create command
	args := []string{"pr", "create"}

	if cfg.Title != "" {
		args = append(args, "--title", cfg.Title)
	}

	if cfg.Base != "" {
		args = append(args, "--base", cfg.Base)
	}

	if cfg.Body != "" {
		args = append(args, "--body", cfg.Body)
	}

	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "already exists") {
			return nil, fmt.Errorf("a PR for this branch already exists")
		}
		return nil, fmt.Errorf("failed to create PR: %s", errMsg)
	}

	// Parse the output to get the PR URL
	prURL := strings.TrimSpace(stdout.String())
	if prURL == "" {
		return nil, fmt.Errorf("no PR URL returned from gh")
	}

	return &PRResult{
		URL: prURL,
	}, nil
}

// GetCurrentBranch returns the name of the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch returns the default branch of the repository
func GetDefaultBranch() (string, error) {
	// Try to get from remote HEAD
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		// Output is like "refs/remotes/origin/main"
		ref := strings.TrimSpace(string(output))
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check for common default branch names
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
		if cmd.Run() == nil {
			return branch, nil
		}
	}

	return "main", nil // Default to main if nothing found
}

// GetCommitsSinceBase returns a summary of commits on the current branch since diverging from base
func GetCommitsSinceBase(baseBranch string) ([]string, error) {
	// Get commits that are on current branch but not on base
	cmd := exec.Command("git", "log", "--oneline", baseBranch+"..HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

// IsBranchPushed checks if the current branch has been pushed to remote
func IsBranchPushed() bool {
	branch, err := GetCurrentBranch()
	if err != nil {
		return false
	}

	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branch)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// PushBranch pushes the current branch to origin
func PushBranch() error {
	branch, err := GetCurrentBranch()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "push", "-u", "origin", branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch: %s", stderr.String())
	}

	return nil
}
