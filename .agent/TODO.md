# Code Review

## Issues Found

- [ ] internal/metrics/collector_test.go:343-364 - Data Race Risk: TestCollector_ConcurrentUpdates writes to collector.currentPending and collector.currentComplete from multiple goroutines without synchronization. The underlying UpdateTodoCounts method lacks thread-safety. Consider adding mutex protection in the Collector struct or documenting that UpdateTodoCounts is not thread-safe.

- [ ] internal/runner/runner_test.go:467-486 - Unused MockTracker: The MockTracker struct is defined but never used in any test. This appears to be dead code that should either be removed or tests should be added that use it.

- [ ] internal/runner/runner_test.go - Missing t.Parallel(): TestGeneratePRTitle, TestGeneratePRBody, TestCopyFile, and other tests don't use t.Parallel() while some tests in the same file do. Consider adding t.Parallel() to all table-driven tests that don't modify shared state for faster test execution and consistency.

- [ ] internal/git/tracker_test.go, internal/git/pr_test.go, internal/worktree/worktree_test.go - Code Duplication: The helper functions setupTestGitRepo() and makeCommit() are duplicated across multiple test files. Consider extracting these to a shared testutil package to reduce duplication and improve maintainability.

- [ ] internal/slack/client_test.go:708-720 - Inefficient containsSubstring Helper: The containsSubstring helper function reimplements strings.Contains with a custom containsHelper. This should simply use strings.Contains from the standard library for clarity and correctness.

- [ ] internal/claude/client_test.go - Resource Leak Risk: Tests like TestClient_WithContext_Integration, TestClient_Kill_RunningProcess, and TestClient_ContextCancellation create real exec.Command processes but some test paths may not properly clean up the process (e.g., if assertions fail before cleanup). Consider using t.Cleanup() for more robust resource management.

- [ ] internal/runner/runner_test.go - Test File Path Changes Current Directory: Tests like TestRunCleanupPhase_WithPatterns use os.Chdir to change directories and rely on defer to restore. If a test panics, this could affect other tests. Consider using t.Chdir() (Go 1.24+) or restructuring tests to avoid directory changes.

- [ ] internal/runner/runner_test.go:1330-1381 - Platform-Specific Test Assumption: TestRunCleanupPhase_FileRemovalError sets file to read-only (0444) expecting removal to fail, but this behavior varies by OS and user permissions (root can still delete). The test correctly handles this with "may or may not produce error" but could be marked with build tags or skip conditions for clarity.
