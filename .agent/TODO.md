# Code Review

## Issues Found

- [ ] internal/metrics/collector_test.go:343-364 - Data Race Risk: TestCollector_ConcurrentUpdates writes to collector.currentPending and collector.currentComplete from multiple goroutines without synchronization. The underlying UpdateTodoCounts method lacks thread-safety. Consider adding mutex protection in the Collector struct or documenting that UpdateTodoCounts is not thread-safe.

- [ ] internal/runner/runner_test.go:467-486 - Unused MockTracker: The MockTracker struct is defined but never used in any test. This appears to be dead code that should either be removed or tests should be added that use it.
