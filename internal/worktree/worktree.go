package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hev/ralph/internal/git"
)

// Manager handles git worktree operations
type Manager struct {
	BaseDir      string
	BranchPrefix string
	Cleanup      bool
	OriginalDir  string
	WorktreePath string
	BranchName   string
}

// NewManager creates a new worktree manager with the given configuration
func NewManager(baseDir, branchPrefix string, cleanup bool) *Manager {
	return &Manager{
		BaseDir:      baseDir,
		BranchPrefix: branchPrefix,
		Cleanup:      cleanup,
	}
}

// Create creates a new worktree and returns its path
// If branch is empty, generates a branch name using the prefix and timestamp
func (m *Manager) Create(branch string) (string, error) {
	// Check if we're in a git repo
	if !git.IsGitRepo() {
		return "", fmt.Errorf("not a git repository")
	}

	// Store original directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	m.OriginalDir = originalDir

	// Generate branch name if not provided
	if branch == "" {
		branch = m.generateBranchName()
	}
	m.BranchName = branch

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(m.BaseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create base directory: %w", err)
	}

	// Sanitize branch name for directory path
	dirName := sanitizeBranchName(branch)
	worktreePath := filepath.Join(m.BaseDir, dirName)
	m.WorktreePath = worktreePath

	// Check if branch exists
	branchExists := m.branchExists(branch)

	var cmd *exec.Cmd
	if branchExists {
		// Use existing branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	} else {
		// Create new branch
		cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %s: %w", string(output), err)
	}

	return worktreePath, nil
}

// Remove removes the worktree and optionally the branch
func (m *Manager) Remove() error {
	if m.WorktreePath == "" {
		return nil
	}

	// Change back to original directory first
	if m.OriginalDir != "" {
		if err := os.Chdir(m.OriginalDir); err != nil {
			return fmt.Errorf("failed to change to original directory: %w", err)
		}
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", "--force", m.WorktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s: %w", string(output), err)
	}

	return nil
}

// Push pushes the worktree branch to the remote
func (m *Manager) Push() error {
	if m.BranchName == "" {
		return fmt.Errorf("no branch name set")
	}

	// Push the branch to origin
	cmd := exec.Command("git", "push", "-u", "origin", m.BranchName)
	cmd.Dir = m.WorktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch: %s: %w", string(output), err)
	}

	return nil
}

// GetBranchName returns the branch name used by the worktree
func (m *Manager) GetBranchName() string {
	return m.BranchName
}

// GetWorktreePath returns the path to the worktree
func (m *Manager) GetWorktreePath() string {
	return m.WorktreePath
}

// generateBranchName creates a branch name using the prefix and timestamp
func (m *Manager) generateBranchName() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s%s", m.BranchPrefix, timestamp)
}

// branchExists checks if a branch already exists
func (m *Manager) branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	err := cmd.Run()
	return err == nil
}

// sanitizeBranchName converts a branch name to a safe directory name
func sanitizeBranchName(branch string) string {
	// Replace slashes with dashes for directory name
	return strings.ReplaceAll(branch, "/", "-")
}

// List returns a list of all worktrees
func List() ([]string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var paths []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			paths = append(paths, path)
		}
	}

	return paths, nil
}
