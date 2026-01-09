package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestGitRepo creates a temporary git repository for testing
func setupTestGitRepo(t *testing.T) string {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
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

func TestNewTracker(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v, want nil", err)
	}

	if tracker == nil {
		t.Fatal("NewTracker() returned nil tracker")
	}

	// Initial delta should be 0
	delta, err := tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v, want nil", err)
	}
	if delta != 0 {
		t.Errorf("CommitsDelta() = %d, want 0", delta)
	}
}

func TestTracker_CommitsDelta(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// No new commits yet
	delta, err := tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v", err)
	}
	if delta != 0 {
		t.Errorf("CommitsDelta() = %d, want 0", delta)
	}

	// Make some commits
	makeCommit(t, "second")
	makeCommit(t, "third")

	// Check delta
	delta, err = tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v", err)
	}
	if delta != 2 {
		t.Errorf("CommitsDelta() = %d, want 2", delta)
	}
}

func TestTracker_CommitsDelta_ZeroCommits(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// No new commits - delta should be 0
	delta, err := tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v", err)
	}
	if delta != 0 {
		t.Errorf("CommitsDelta() = %d, want 0", delta)
	}
}

func TestTracker_CommitsDelta_FiveCommits(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// Make 5 commits with unique names
	for i := 1; i <= 5; i++ {
		makeCommit(t, "commit-"+string(rune('a'+i-1)))
	}

	delta, err := tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v", err)
	}
	if delta != 5 {
		t.Errorf("CommitsDelta() = %d, want 5", delta)
	}
}

func TestTracker_UpdateBaseline(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// Make some commits
	makeCommit(t, "second")
	makeCommit(t, "third")

	// Delta should be 2
	delta, _ := tracker.CommitsDelta()
	if delta != 2 {
		t.Errorf("CommitsDelta() = %d, want 2", delta)
	}

	// Update baseline
	if err := tracker.UpdateBaseline(); err != nil {
		t.Fatalf("UpdateBaseline() error = %v", err)
	}

	// Delta should reset to 0
	delta, _ = tracker.CommitsDelta()
	if delta != 0 {
		t.Errorf("CommitsDelta() after UpdateBaseline() = %d, want 0", delta)
	}

	// Make another commit
	makeCommit(t, "fourth")

	// Delta should be 1
	delta, _ = tracker.CommitsDelta()
	if delta != 1 {
		t.Errorf("CommitsDelta() = %d, want 1", delta)
	}
}

func TestGetCurrentCommitCount(t *testing.T) {
	setupTestGitRepo(t)

	// No commits yet - should return 0
	count, err := GetCurrentCommitCount()
	if err != nil {
		t.Fatalf("GetCurrentCommitCount() error = %v", err)
	}
	if count != 0 {
		t.Errorf("GetCurrentCommitCount() = %d, want 0 (no commits)", count)
	}

	// Make commits and check count
	makeCommit(t, "first")
	count, err = GetCurrentCommitCount()
	if err != nil {
		t.Fatalf("GetCurrentCommitCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("GetCurrentCommitCount() = %d, want 1", count)
	}

	makeCommit(t, "second")
	makeCommit(t, "third")
	count, err = GetCurrentCommitCount()
	if err != nil {
		t.Fatalf("GetCurrentCommitCount() error = %v", err)
	}
	if count != 3 {
		t.Errorf("GetCurrentCommitCount() = %d, want 3", count)
	}
}

func TestIsGitRepo(t *testing.T) {
	// Test in a git repo
	setupTestGitRepo(t)

	if !IsGitRepo() {
		t.Error("IsGitRepo() = false, want true in git repo")
	}
}

func TestIsGitRepo_NotARepo(t *testing.T) {
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

	if IsGitRepo() {
		t.Error("IsGitRepo() = true, want false in non-git directory")
	}
}

func TestTracker_ImplementsCommitTracker(t *testing.T) {
	// Compile-time check that Tracker implements CommitTracker
	var _ CommitTracker = (*Tracker)(nil)
}

func TestGetCommitCount_EmptyRepo(t *testing.T) {
	setupTestGitRepo(t)

	// Empty repo (no commits) should return 0, not an error
	count, err := getCommitCount()
	if err != nil {
		t.Fatalf("getCommitCount() error = %v, want nil", err)
	}
	if count != 0 {
		t.Errorf("getCommitCount() = %d, want 0 for empty repo", count)
	}
}

func TestGetCommitCount_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Not a git repo - should return 0 without error
	count, err := getCommitCount()
	if err != nil {
		t.Fatalf("getCommitCount() error = %v, want nil", err)
	}
	if count != 0 {
		t.Errorf("getCommitCount() = %d, want 0 for non-git repo", count)
	}
}

func TestNewTracker_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// NewTracker should work (returns 0 count) even in non-git repo
	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v, want nil", err)
	}
	if tracker == nil {
		t.Fatal("NewTracker() returned nil tracker")
	}
}

func TestTracker_CommitsDelta_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create tracker in non-git directory
	tracker, _ := NewTracker()

	// CommitsDelta should return 0 without error
	delta, err := tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v, want nil", err)
	}
	if delta != 0 {
		t.Errorf("CommitsDelta() = %d, want 0", delta)
	}
}

func TestTracker_UpdateBaseline_NotGitRepo(t *testing.T) {
	// Create a non-git directory
	tmpDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create tracker in non-git directory
	tracker, _ := NewTracker()

	// UpdateBaseline should work without error
	err = tracker.UpdateBaseline()
	if err != nil {
		t.Fatalf("UpdateBaseline() error = %v, want nil", err)
	}
}

func TestTracker_CommitsDelta_NegativeDelta(t *testing.T) {
	setupTestGitRepo(t)

	// Make several commits
	makeCommit(t, "initial")
	makeCommit(t, "second")
	makeCommit(t, "third")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// Now reset to remove commits (simulating negative delta scenario)
	// Reset to first commit
	exec.Command("git", "reset", "--hard", "HEAD~2").Run()

	// Delta should be 0 (not negative) since code handles this case
	delta, err := tracker.CommitsDelta()
	if err != nil {
		t.Fatalf("CommitsDelta() error = %v", err)
	}
	if delta != 0 {
		t.Errorf("CommitsDelta() = %d, want 0 for negative delta case", delta)
	}
}

func TestTracker_MultipleIterations(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	// Simulate iteration 1: make 2 commits
	makeCommit(t, "iter1-a")
	makeCommit(t, "iter1-b")

	delta, _ := tracker.CommitsDelta()
	if delta != 2 {
		t.Errorf("After iteration 1: CommitsDelta() = %d, want 2", delta)
	}

	// Update baseline
	tracker.UpdateBaseline()

	// Simulate iteration 2: make 3 commits
	makeCommit(t, "iter2-a")
	makeCommit(t, "iter2-b")
	makeCommit(t, "iter2-c")

	delta, _ = tracker.CommitsDelta()
	if delta != 3 {
		t.Errorf("After iteration 2: CommitsDelta() = %d, want 3", delta)
	}

	// Update baseline again
	tracker.UpdateBaseline()

	// No new commits
	delta, _ = tracker.CommitsDelta()
	if delta != 0 {
		t.Errorf("After no new commits: CommitsDelta() = %d, want 0", delta)
	}
}

func TestTracker_DirectFieldAccess(t *testing.T) {
	setupTestGitRepo(t)

	// Make initial commit
	makeCommit(t, "initial")

	tracker, _ := NewTracker()

	// Access the initial count through the struct
	if tracker.initialCommitCount < 0 {
		t.Errorf("initialCommitCount = %d, want >= 0", tracker.initialCommitCount)
	}
}
