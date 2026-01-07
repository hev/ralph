package git

import (
	"os/exec"
	"strconv"
	"strings"
)

// Tracker tracks git commits during a session
type Tracker struct {
	initialCommitCount int
}

// NewTracker creates a new git tracker, recording the initial commit count
func NewTracker() (*Tracker, error) {
	count, err := getCommitCount()
	if err != nil {
		return nil, err
	}

	return &Tracker{
		initialCommitCount: count,
	}, nil
}

// CommitsDelta returns the number of commits made since the tracker was created
func (t *Tracker) CommitsDelta() (int, error) {
	count, err := getCommitCount()
	if err != nil {
		return 0, err
	}

	delta := count - t.initialCommitCount
	if delta < 0 {
		// This could happen if commits were removed (rebase, reset, etc.)
		return 0, nil
	}
	return delta, nil
}

// UpdateBaseline updates the initial commit count to the current count
// Useful for tracking commits per iteration
func (t *Tracker) UpdateBaseline() error {
	count, err := getCommitCount()
	if err != nil {
		return err
	}
	t.initialCommitCount = count
	return nil
}

// getCommitCount returns the total number of commits in the current branch
func getCommitCount() (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Not a git repo or no commits yet
		return 0, nil
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, nil
	}

	return count, nil
}

// GetCurrentCommitCount returns the current commit count (utility function)
func GetCurrentCommitCount() (int, error) {
	return getCommitCount()
}

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}
