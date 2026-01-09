package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestGitRepo creates a temporary git repository for testing
func setupTestGitRepo(t *testing.T) string {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to get current dir: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "config", "user.email", "test@test.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Register cleanup
	t.Cleanup(func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

// makeCommit creates a commit in the test repo
func makeCommit(t *testing.T, message string) {
	t.Helper()

	// Create a file to commit
	filename := filepath.Join(".", message+".txt")
	if err := os.WriteFile(filename, []byte(message), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Add and commit
	if err := exec.Command("git", "add", ".").Run(); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name         string
		baseDir      string
		branchPrefix string
		cleanup      bool
	}{
		{
			name:         "with defaults",
			baseDir:      "/tmp/worktrees",
			branchPrefix: "ralph/",
			cleanup:      false,
		},
		{
			name:         "with cleanup enabled",
			baseDir:      "/var/worktrees",
			branchPrefix: "feature/",
			cleanup:      true,
		},
		{
			name:         "with empty prefix",
			baseDir:      "/tmp/wt",
			branchPrefix: "",
			cleanup:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.baseDir, tt.branchPrefix, tt.cleanup)

			if m == nil {
				t.Fatal("NewManager() returned nil")
			}
			if m.BaseDir != tt.baseDir {
				t.Errorf("BaseDir = %q, want %q", m.BaseDir, tt.baseDir)
			}
			if m.BranchPrefix != tt.branchPrefix {
				t.Errorf("BranchPrefix = %q, want %q", m.BranchPrefix, tt.branchPrefix)
			}
			if m.Cleanup != tt.cleanup {
				t.Errorf("Cleanup = %v, want %v", m.Cleanup, tt.cleanup)
			}
			// Initial values should be empty
			if m.OriginalDir != "" {
				t.Errorf("OriginalDir = %q, want empty", m.OriginalDir)
			}
			if m.WorktreePath != "" {
				t.Errorf("WorktreePath = %q, want empty", m.WorktreePath)
			}
			if m.BranchName != "" {
				t.Errorf("BranchName = %q, want empty", m.BranchName)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "no slashes",
			branch:   "feature-branch",
			expected: "feature-branch",
		},
		{
			name:     "single slash",
			branch:   "feature/branch",
			expected: "feature-branch",
		},
		{
			name:     "multiple slashes",
			branch:   "feature/sub/branch",
			expected: "feature-sub-branch",
		},
		{
			name:     "ralph prefix",
			branch:   "ralph/20240101-120000",
			expected: "ralph-20240101-120000",
		},
		{
			name:     "leading slash",
			branch:   "/branch",
			expected: "-branch",
		},
		{
			name:     "trailing slash",
			branch:   "branch/",
			expected: "branch-",
		},
		{
			name:     "only slashes",
			branch:   "///",
			expected: "---",
		},
		{
			name:     "empty string",
			branch:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeBranchName(tt.branch)
			if result != tt.expected {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.branch, result, tt.expected)
			}
		})
	}
}

func TestManager_generateBranchName(t *testing.T) {
	tests := []struct {
		name         string
		branchPrefix string
	}{
		{
			name:         "ralph prefix",
			branchPrefix: "ralph/",
		},
		{
			name:         "feature prefix",
			branchPrefix: "feature/",
		},
		{
			name:         "empty prefix",
			branchPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager("/tmp", tt.branchPrefix, false)

			branchName := m.generateBranchName()

			// Check prefix
			if !strings.HasPrefix(branchName, tt.branchPrefix) {
				t.Errorf("generateBranchName() = %q, want prefix %q", branchName, tt.branchPrefix)
			}

			// Extract timestamp part
			timestamp := strings.TrimPrefix(branchName, tt.branchPrefix)

			// Verify format: YYYYMMDD-HHMMSS
			if len(timestamp) != 15 { // 8 digits + dash + 6 digits
				t.Errorf("timestamp part %q has wrong length, want 15 chars", timestamp)
			}

			// Verify the timestamp can be parsed (just check format validity)
			_, err := time.Parse("20060102-150405", timestamp)
			if err != nil {
				t.Errorf("failed to parse timestamp %q: %v", timestamp, err)
			}

			// Verify it looks like a recent date (current year)
			currentYear := time.Now().Format("2006")
			if !strings.HasPrefix(timestamp, currentYear) {
				t.Errorf("timestamp %q doesn't start with current year %s", timestamp, currentYear)
			}
		})
	}
}

func TestManager_GetBranchName(t *testing.T) {
	m := NewManager("/tmp", "ralph/", false)

	// Initially empty
	if got := m.GetBranchName(); got != "" {
		t.Errorf("GetBranchName() = %q, want empty initially", got)
	}

	// Set branch name
	m.BranchName = "test-branch"
	if got := m.GetBranchName(); got != "test-branch" {
		t.Errorf("GetBranchName() = %q, want %q", got, "test-branch")
	}
}

func TestManager_GetWorktreePath(t *testing.T) {
	m := NewManager("/tmp", "ralph/", false)

	// Initially empty
	if got := m.GetWorktreePath(); got != "" {
		t.Errorf("GetWorktreePath() = %q, want empty initially", got)
	}

	// Set worktree path
	m.WorktreePath = "/tmp/worktree-path"
	if got := m.GetWorktreePath(); got != "/tmp/worktree-path" {
		t.Errorf("GetWorktreePath() = %q, want %q", got, "/tmp/worktree-path")
	}
}

func TestManager_Create_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(origDir)

	m := NewManager(filepath.Join(tmpDir, "worktrees"), "ralph/", false)
	_, err = m.Create("")

	if err == nil {
		t.Error("Create() in non-git repo should return error")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Create() error = %q, want error containing 'not a git repository'", err.Error())
	}
}

func TestManager_Create_WithGeneratedBranch(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	worktreeBase := filepath.Join(repoDir, "worktrees")
	m := NewManager(worktreeBase, "ralph/", false)

	path, err := m.Create("")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Cleanup worktree
	defer func() {
		os.Chdir(repoDir)
		exec.Command("git", "worktree", "remove", "--force", path).Run()
	}()

	// Verify path was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Create() did not create worktree directory")
	}

	// Verify branch name was set
	if m.BranchName == "" {
		t.Error("Create() did not set BranchName")
	}
	if !strings.HasPrefix(m.BranchName, "ralph/") {
		t.Errorf("BranchName = %q, want prefix 'ralph/'", m.BranchName)
	}

	// Verify worktree path was set
	if m.WorktreePath != path {
		t.Errorf("WorktreePath = %q, want %q", m.WorktreePath, path)
	}

	// Verify original directory was stored
	if m.OriginalDir == "" {
		t.Error("Create() did not set OriginalDir")
	}
}

func TestManager_Create_WithCustomBranch(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	worktreeBase := filepath.Join(repoDir, "worktrees")
	m := NewManager(worktreeBase, "ralph/", false)

	customBranch := "feature/custom-branch"
	path, err := m.Create(customBranch)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Cleanup worktree
	defer func() {
		os.Chdir(repoDir)
		exec.Command("git", "worktree", "remove", "--force", path).Run()
	}()

	// Verify branch name was set to custom value
	if m.BranchName != customBranch {
		t.Errorf("BranchName = %q, want %q", m.BranchName, customBranch)
	}

	// Verify path uses sanitized branch name
	expectedPath := filepath.Join(worktreeBase, "feature-custom-branch")
	if m.WorktreePath != expectedPath {
		t.Errorf("WorktreePath = %q, want %q", m.WorktreePath, expectedPath)
	}
}

func TestManager_Create_ExistingBranch(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Create a branch first
	existingBranch := "existing-branch"
	if err := exec.Command("git", "branch", existingBranch).Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	worktreeBase := filepath.Join(repoDir, "worktrees")
	m := NewManager(worktreeBase, "ralph/", false)

	path, err := m.Create(existingBranch)
	if err != nil {
		t.Fatalf("Create() with existing branch error = %v", err)
	}

	// Cleanup worktree
	defer func() {
		os.Chdir(repoDir)
		exec.Command("git", "worktree", "remove", "--force", path).Run()
	}()

	// Verify branch name was set
	if m.BranchName != existingBranch {
		t.Errorf("BranchName = %q, want %q", m.BranchName, existingBranch)
	}
}

func TestManager_branchExists(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	m := NewManager("/tmp", "ralph/", false)

	// Main branch should exist (master or main depending on git config)
	// Create a known branch to test
	testBranch := "test-branch-exists"
	if err := exec.Command("git", "branch", testBranch).Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	if !m.branchExists(testBranch) {
		t.Errorf("branchExists(%q) = false, want true", testBranch)
	}

	// Non-existent branch
	if m.branchExists("non-existent-branch-xyz") {
		t.Error("branchExists('non-existent-branch-xyz') = true, want false")
	}
}

func TestManager_Remove_NoWorktree(t *testing.T) {
	m := NewManager("/tmp", "ralph/", false)

	// Remove with no worktree path should be a no-op
	err := m.Remove()
	if err != nil {
		t.Errorf("Remove() with no worktree error = %v, want nil", err)
	}
}

func TestManager_Remove_WithWorktree(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	worktreeBase := filepath.Join(repoDir, "worktrees")
	m := NewManager(worktreeBase, "ralph/", false)

	// Create a worktree first
	path, err := m.Create("")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("worktree was not created")
	}

	// Change to worktree directory (simulating working in it)
	if err := os.Chdir(path); err != nil {
		t.Fatalf("failed to change to worktree: %v", err)
	}

	// Remove the worktree
	err = m.Remove()
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Remove() did not remove worktree directory")
	}
}

func TestManager_Push_NoBranch(t *testing.T) {
	m := NewManager("/tmp", "ralph/", false)

	err := m.Push()
	if err == nil {
		t.Error("Push() with no branch should return error")
	}
	if !strings.Contains(err.Error(), "no branch name set") {
		t.Errorf("Push() error = %q, want error containing 'no branch name set'", err.Error())
	}
}

func TestList_InGitRepo(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	// List worktrees (should at least have the main worktree)
	paths, err := List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Should have at least one worktree (the main repo)
	if len(paths) < 1 {
		t.Error("List() returned no worktrees, want at least 1")
	}

	// First worktree should be the main repo
	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resolvedRepoDir, _ := filepath.EvalSymlinks(repoDir)
	found := false
	for _, p := range paths {
		resolvedP, _ := filepath.EvalSymlinks(p)
		if resolvedP == resolvedRepoDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() did not include main repo path %q in %v", repoDir, paths)
	}
}

func TestList_WithMultipleWorktrees(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	worktreeBase := filepath.Join(repoDir, "worktrees")
	m := NewManager(worktreeBase, "ralph/", false)

	// Create a worktree
	path, err := m.Create("test-wt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer func() {
		os.Chdir(repoDir)
		exec.Command("git", "worktree", "remove", "--force", path).Run()
	}()

	// List worktrees
	paths, err := List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Should have at least 2 worktrees (main + created one)
	if len(paths) < 2 {
		t.Errorf("List() returned %d worktrees, want at least 2", len(paths))
	}

	// Verify our created worktree is in the list
	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resolvedPath, _ := filepath.EvalSymlinks(path)
	found := false
	for _, p := range paths {
		resolvedP, _ := filepath.EvalSymlinks(p)
		if resolvedP == resolvedPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() did not include created worktree path %q in %v", path, paths)
	}
}

func TestList_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(origDir)

	// List should fail in non-git repo
	_, err = List()
	if err == nil {
		t.Error("List() in non-git repo should return error")
	}
}

func TestManager_Create_CreatesBaseDir(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Use a non-existent base directory
	worktreeBase := filepath.Join(repoDir, "non-existent", "nested", "worktrees")
	m := NewManager(worktreeBase, "ralph/", false)

	path, err := m.Create("")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Cleanup worktree
	defer func() {
		os.Chdir(repoDir)
		exec.Command("git", "worktree", "remove", "--force", path).Run()
	}()

	// Verify base directory was created
	if _, err := os.Stat(worktreeBase); os.IsNotExist(err) {
		t.Error("Create() did not create base directory")
	}
}

func TestManager_FullWorkflow(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	makeCommit(t, "initial")

	worktreeBase := filepath.Join(repoDir, "worktrees")
	m := NewManager(worktreeBase, "ralph/", true)

	// Create worktree
	path, err := m.Create("")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify getters work
	if m.GetBranchName() == "" {
		t.Error("GetBranchName() is empty after Create()")
	}
	if m.GetWorktreePath() != path {
		t.Errorf("GetWorktreePath() = %q, want %q", m.GetWorktreePath(), path)
	}

	// Change to worktree and verify it's a git repo
	if err := os.Chdir(path); err != nil {
		t.Fatalf("failed to change to worktree: %v", err)
	}

	// Make a commit in the worktree
	makeCommit(t, "worktree-commit")

	// Remove the worktree
	err = m.Remove()
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Remove() did not remove worktree directory")
	}
}
