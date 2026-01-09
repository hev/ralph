# Code Review

## Issues Found

- [ ] internal/metrics/collector_test.go:343-364 - Data Race Risk: TestCollector_ConcurrentUpdates writes to collector.currentPending and collector.currentComplete from multiple goroutines without synchronization. The underlying UpdateTodoCounts method lacks thread-safety. Consider adding mutex protection in the Collector struct or documenting that UpdateTodoCounts is not thread-safe.

- [ ] internal/runner/runner_test.go:467-486 - Unused MockTracker: The MockTracker struct is defined but never used in any test. This appears to be dead code that should either be removed or tests should be added that use it.

- [ ] internal/runner/runner_test.go - Missing t.Parallel(): TestGeneratePRTitle, TestGeneratePRBody, TestCopyFile, and other tests don't use t.Parallel() while some tests in the same file do. Consider adding t.Parallel() to all table-driven tests that don't modify shared state for faster test execution and consistency.

- [ ] internal/git/tracker_test.go, internal/git/pr_test.go, internal/worktree/worktree_test.go - Code Duplication: The helper functions setupTestGitRepo() and makeCommit() are duplicated across multiple test files. Consider extracting these to a shared testutil package to reduce duplication and improve maintainability.
