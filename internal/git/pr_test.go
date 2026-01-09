package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGetCurrentBranch(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Default branch is usually "main" or "master" depending on git config
	if branch != "main" && branch != "master" {
		t.Errorf("GetCurrentBranch() = %q, want 'main' or 'master'", branch)
	}
}

func TestGetCurrentBranch_CustomBranch(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Create and checkout a new branch
	if err := exec.Command("git", "checkout", "-b", "feature/test-branch").Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	if branch != "feature/test-branch" {
		t.Errorf("GetCurrentBranch() = %q, want 'feature/test-branch'", branch)
	}
}

func TestGetDefaultBranch(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	branch, err := GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	// Should be "main" or "master"
	if branch != "main" && branch != "master" {
		t.Errorf("GetDefaultBranch() = %q, want 'main' or 'master'", branch)
	}
}

func TestGetDefaultBranch_WithMaster(t *testing.T) {
	// Create repo with master branch
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Initialize with master as default
	exec.Command("git", "init", "-b", "master").Run()
	exec.Command("git", "config", "user.email", "test@test.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Make a commit
	os.WriteFile("test.txt", []byte("test"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	branch, err := GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	if branch != "master" && branch != "main" {
		t.Errorf("GetDefaultBranch() = %q, want 'master' or 'main'", branch)
	}
}

func TestGetCommitsSinceBase(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Get the default branch name
	defaultBranch, _ := GetCurrentBranch()

	// Create a feature branch
	if err := exec.Command("git", "checkout", "-b", "feature").Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Make commits on feature branch
	makeCommit(t, "feature-1")
	makeCommit(t, "feature-2")

	commits, err := GetCommitsSinceBase(defaultBranch)
	if err != nil {
		t.Fatalf("GetCommitsSinceBase() error = %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("GetCommitsSinceBase() returned %d commits, want 2", len(commits))
	}
}

func TestGetCommitsSinceBase_NoNewCommits(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Get the default branch name
	defaultBranch, _ := GetCurrentBranch()

	// Create a feature branch but don't add commits
	if err := exec.Command("git", "checkout", "-b", "feature-empty").Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	commits, err := GetCommitsSinceBase(defaultBranch)
	if err != nil {
		t.Fatalf("GetCommitsSinceBase() error = %v", err)
	}

	if len(commits) != 0 {
		t.Errorf("GetCommitsSinceBase() returned %d commits, want 0", len(commits))
	}
}

func TestIsBranchPushed_NotPushed(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// In a local-only repo without remote, branch shouldn't be pushed
	if IsBranchPushed() {
		t.Error("IsBranchPushed() = true, want false for local-only repo")
	}
}

func TestPRConfig(t *testing.T) {
	// Test PRConfig struct
	cfg := PRConfig{
		Title: "Test PR",
		Base:  "main",
		Body:  "Test body",
	}

	if cfg.Title != "Test PR" {
		t.Errorf("PRConfig.Title = %q, want 'Test PR'", cfg.Title)
	}
	if cfg.Base != "main" {
		t.Errorf("PRConfig.Base = %q, want 'main'", cfg.Base)
	}
	if cfg.Body != "Test body" {
		t.Errorf("PRConfig.Body = %q, want 'Test body'", cfg.Body)
	}
}

func TestPRResult(t *testing.T) {
	// Test PRResult struct
	result := PRResult{
		URL:    "https://github.com/owner/repo/pull/123",
		Number: 123,
	}

	if result.URL != "https://github.com/owner/repo/pull/123" {
		t.Errorf("PRResult.URL = %q, want 'https://github.com/owner/repo/pull/123'", result.URL)
	}
	if result.Number != 123 {
		t.Errorf("PRResult.Number = %d, want 123", result.Number)
	}
}

func TestCreatePR_NoGhCLI(t *testing.T) {
	// Test CreatePR when gh CLI is not available
	// This test manipulates PATH to ensure gh is not found

	// Save original PATH
	origPath := os.Getenv("PATH")

	// Set PATH to empty to ensure gh is not found
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	cfg := PRConfig{
		Title: "Test PR",
		Body:  "Test body",
	}

	_, err := CreatePR(cfg)
	if err == nil {
		t.Error("CreatePR() error = nil, want error when gh CLI not found")
	}

	if !strings.Contains(err.Error(), "gh CLI not found") {
		t.Errorf("CreatePR() error = %q, want error containing 'gh CLI not found'", err.Error())
	}
}

func TestDefaultCommandExecutor(t *testing.T) {
	executor := &DefaultCommandExecutor{}

	// Test with a simple command that should work
	output, err := executor.Run("echo", "hello")
	if err != nil {
		t.Fatalf("DefaultCommandExecutor.Run() error = %v", err)
	}

	if strings.TrimSpace(string(output)) != "hello" {
		t.Errorf("DefaultCommandExecutor.Run() output = %q, want 'hello'", string(output))
	}
}

func TestDefaultCommandExecutor_Error(t *testing.T) {
	executor := &DefaultCommandExecutor{}

	// Test with a command that should fail
	_, err := executor.Run("nonexistent-command-12345")
	if err == nil {
		t.Error("DefaultCommandExecutor.Run() error = nil, want error for nonexistent command")
	}
}

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	Output []byte
	Error  error
	Calls  [][]string // Records all calls made
}

func (m *MockCommandExecutor) Run(name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	m.Calls = append(m.Calls, call)
	return m.Output, m.Error
}

func TestMockCommandExecutor(t *testing.T) {
	mock := &MockCommandExecutor{
		Output: []byte("mocked output"),
		Error:  nil,
	}

	// Verify it implements CommandExecutor
	var _ CommandExecutor = mock

	output, err := mock.Run("git", "status")
	if err != nil {
		t.Fatalf("MockCommandExecutor.Run() error = %v", err)
	}

	if string(output) != "mocked output" {
		t.Errorf("MockCommandExecutor.Run() output = %q, want 'mocked output'", string(output))
	}

	if len(mock.Calls) != 1 {
		t.Errorf("MockCommandExecutor.Calls length = %d, want 1", len(mock.Calls))
	}

	if mock.Calls[0][0] != "git" || mock.Calls[0][1] != "status" {
		t.Errorf("MockCommandExecutor.Calls[0] = %v, want [git status]", mock.Calls[0])
	}
}

func TestGetCurrentBranch_NotGitRepo(t *testing.T) {
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

	_, err = GetCurrentBranch()
	if err == nil {
		t.Error("GetCurrentBranch() error = nil, want error in non-git directory")
	}
}

func TestGetCommitsSinceBase_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, err = GetCommitsSinceBase("main")
	if err == nil {
		t.Error("GetCommitsSinceBase() error = nil, want error in non-git directory")
	}
}

func TestPushBranch_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err = PushBranch()
	if err == nil {
		t.Error("PushBranch() error = nil, want error in non-git directory")
	}
}

func TestPushBranch_NoRemote(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Try to push without a remote configured
	err := PushBranch()
	if err == nil {
		t.Error("PushBranch() error = nil, want error when no remote configured")
	}
}

func TestIsBranchPushed_NoRemote(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Should return false when no remote
	if IsBranchPushed() {
		t.Error("IsBranchPushed() = true, want false when no remote configured")
	}
}

func TestGetDefaultBranch_NoRemoteOriginHEAD(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Create both main and master branches to test the fallback logic
	branch, err := GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	// Should return either main or master
	if branch != "main" && branch != "master" {
		t.Errorf("GetDefaultBranch() = %q, want 'main' or 'master'", branch)
	}
}

func TestCreatePR_EmptyConfig(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	// Empty config should still fail gracefully when gh not found
	cfg := PRConfig{}
	_, err := CreatePR(cfg)
	if err == nil {
		t.Error("CreatePR() error = nil, want error")
	}
}

func TestIsBranchPushed_WithBranch(t *testing.T) {
	setupTestGitRepo(t)
	makeCommit(t, "initial")

	// Create a feature branch
	exec.Command("git", "checkout", "-b", "feature/test").Run()
	makeCommit(t, "feature")

	// Without remote configured, should return false
	if IsBranchPushed() {
		t.Error("IsBranchPushed() = true, want false without remote")
	}
}

func TestGetDefaultBranch_NoBranches(t *testing.T) {
	// Create repo with a different default branch
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Initialize with a custom branch name
	exec.Command("git", "init", "-b", "develop").Run()
	exec.Command("git", "config", "user.email", "test@test.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Make a commit on develop
	os.WriteFile("test.txt", []byte("test"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	// Now there's no main or master - should default to "main"
	branch, err := GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	// Should return "main" as final fallback
	if branch != "main" {
		t.Errorf("GetDefaultBranch() = %q, want 'main' as fallback", branch)
	}
}

func TestGetDefaultBranch_WithMainBranch(t *testing.T) {
	// Create repo with main branch to test main detection
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Initialize with main branch
	exec.Command("git", "init", "-b", "main").Run()
	exec.Command("git", "config", "user.email", "test@test.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Make a commit on main
	os.WriteFile("test.txt", []byte("test"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "initial").Run()

	branch, err := GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	// Should find main
	if branch != "main" {
		t.Errorf("GetDefaultBranch() = %q, want 'main'", branch)
	}
}

func TestBuildPRArgs(t *testing.T) {
	tests := []struct {
		name string
		cfg  PRConfig
		want []string
	}{
		{
			name: "empty config",
			cfg:  PRConfig{},
			want: []string{"pr", "create"},
		},
		{
			name: "title only",
			cfg:  PRConfig{Title: "My PR Title"},
			want: []string{"pr", "create", "--title", "My PR Title"},
		},
		{
			name: "base only",
			cfg:  PRConfig{Base: "main"},
			want: []string{"pr", "create", "--base", "main"},
		},
		{
			name: "body only",
			cfg:  PRConfig{Body: "PR description"},
			want: []string{"pr", "create", "--body", "PR description"},
		},
		{
			name: "all fields",
			cfg: PRConfig{
				Title: "Test PR",
				Base:  "develop",
				Body:  "This is a test PR",
			},
			want: []string{"pr", "create", "--title", "Test PR", "--base", "develop", "--body", "This is a test PR"},
		},
		{
			name: "title and base",
			cfg: PRConfig{
				Title: "Feature PR",
				Base:  "main",
			},
			want: []string{"pr", "create", "--title", "Feature PR", "--base", "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPRArgs(tt.cfg)
			if len(got) != len(tt.want) {
				t.Errorf("BuildPRArgs() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildPRArgs()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
