# Test Coverage Plan for Ralph

## Current State

The codebase has **zero test coverage**. No `*_test.go` files exist.

## Testing Strategy

Use standard Go testing (`testing` package) with table-driven tests. Create interfaces for external dependencies to enable mocking. Focus on unit tests first, then add integration tests for critical paths.

## Package Testing Priority

### 1. `internal/todo` - Parser (Low Risk, Easy Start)

**File to create:** `internal/todo/parser_test.go`

**Functions to test:**
- `ParseFile()` - File reading and counting
- `ParseItems()` - Full item extraction with text

**Test cases:**
- Pending items: `- [ ] task`
- Completed items: `- [x] task` and `- [X] task`
- In-progress items: `- [-] task` and `- [~] task`
- Mixed checkbox styles in same file
- Empty file
- File with no checkboxes
- Nested lists (should still count)
- Malformed checkboxes
- File not found error

**Approach:** Create test fixture files or use inline strings with `strings.NewReader`.

---

### 2. `internal/config` - Configuration (Medium Complexity)

**File to create:** `internal/config/config_test.go`

**Functions to test:**
- `DefaultConfig()` - Verify all defaults are sensible
- `LoadFromFile()` - YAML loading and merging
- `FindConfigFiles()` - File discovery logic
- `GetSlackNotifyUsers()` - CSV parsing
- `ScratchpadInstructions()` - Template generation
- `CodeReviewInstructions()` - Template generation

**Test cases:**
- Default config values are correct
- YAML overwrites only specified fields
- Missing YAML file returns error
- Invalid YAML returns error
- Empty YAML file works (no changes)
- Partial YAML (some fields only)
- Slack user parsing: single user, multiple users, empty string
- Environment variable `RALPH_MODEL` override
- Config file precedence (global vs local)

**Approach:** Create temp YAML files for file-based tests. Test merging logic with struct comparison.

---

### 3. `internal/claude` - Parser (Critical, Medium Complexity)

**File to create:** `internal/claude/parser_test.go`

**Functions to test:**
- `ParseAndPrint()` - Main dispatcher (with captured output)
- `formatTodoWrite()` - Checklist formatting
- `formatEdit()` - Diff-style output
- `formatReadResult()` - Truncation logic
- `formatChecklist()` - Markdown checklist detection
- `stripSystemReminders()` - Regex filtering

**Test cases:**
- Valid JSON message parsing for each type
- Invalid JSON handling
- System reminder stripping (various formats)
- TodoWrite with pending/completed/in-progress items
- Edit formatting with old/new strings
- Read result truncation at threshold
- Checklist detection (markdown checkboxes)
- Empty content handling
- Very large content truncation

**Approach:** Create JSON fixture strings. Capture stdout for output verification or refactor to return strings.

---

### 4. `internal/claude` - Client (Requires Mocking)

**File to create:** `internal/claude/client_test.go`

**Functions to test:**
- `NewClient()` - Command construction
- `Wait()` - Exit code extraction
- Error message parsing from stderr

**Test cases:**
- Command flags are correctly set
- Exit code 0 handling
- Non-zero exit code handling
- Process kill behavior
- Timeout handling

**Approach:** Create interface for command execution to mock `exec.Cmd`. Alternatively, test command construction without execution.

---

### 5. `internal/git` - Tracker and PR (Requires Mocking)

**Files to create:**
- `internal/git/tracker_test.go`
- `internal/git/pr_test.go`

**Functions to test:**
- `NewTracker()` - Initial baseline
- `CommitsDelta()` - Delta calculation
- `UpdateBaseline()` - Baseline reset
- `CreatePR()` - PR creation
- `GetCurrentBranch()` - Branch detection
- `GetDefaultBranch()` - main/master detection
- `IsBranchPushed()` - Remote check

**Test cases:**
- Commit delta: 0 commits, 5 commits, negative (shouldn't happen)
- Branch detection: main, master, custom default
- PR creation success
- PR creation failure (gh not installed, auth error)
- Already pushed vs needs push

**Approach:** Create command executor interface to mock git/gh CLI calls.

---

### 6. `internal/worktree` - Manager (Requires Mocking)

**File to create:** `internal/worktree/worktree_test.go`

**Functions to test:**
- `Create()` - Worktree creation
- `Remove()` - Cleanup
- `generateBranchName()` - Name generation
- `sanitizeBranchName()` - Path sanitization
- `branchExists()` - Branch check

**Test cases:**
- Branch name format: `ralph/YYYYMMDD-HHMMSS`
- Custom branch name provided
- Branch sanitization: `/` to `-`
- Branch already exists handling
- Worktree path construction
- Remove with force flag
- Directory switching during cleanup

**Approach:** Mock git commands. Test name generation and sanitization without mocking.

---

### 7. `internal/slack` - Messages (Pure Functions, Easy)

**File to create:** `internal/slack/messages_test.go`

**Functions to test:**
- `FormatSessionStart()`
- `FormatSessionEnd()`
- `FormatTodoStarted()`
- `FormatTodoCompleted()`
- `FormatCodeReviewStarted()`
- `FormatCodeReviewComplete()`
- `FormatCleanupStarted()`
- `FormatCleanupComplete()`
- `FormatPRCreated()`
- `formatDuration()`
- `truncateSessionID()`

**Test cases:**
- Duration formatting: 0s, 30s, 90s, 3600s, 7200s
- Session ID truncation: full UUID to 8 chars
- All message types contain required fields
- User mention formatting
- GitHub URL inclusion
- Completion rate calculation (0%, 50%, 100%)

**Approach:** Pure function testing with struct comparison.

---

### 8. `internal/slack` - Client (HTTP Mocking)

**File to create:** `internal/slack/client_test.go`

**Functions to test:**
- `PostWebhook()` - Webhook POST
- `PostMessage()` - API POST
- `PostWithRetry()` - Retry logic
- `IsConfigured()` - Configuration check

**Test cases:**
- Successful webhook post
- Webhook error (4xx, 5xx)
- Successful API post with thread_ts
- API error handling
- Retry on 5xx (1s, 2s, 4s backoff)
- No retry on 4xx
- Context cancellation during retry
- Not configured returns early

**Approach:** Use `httptest.Server` for HTTP mocking.

---

### 9. `internal/slack` - Notifier (Integration)

**File to create:** `internal/slack/notifier_test.go`

**Functions to test:**
- `SessionStart()` / `SessionEnd()`
- `TodoStarted()` / `TodoCompleted()`
- Message routing (webhook vs API)
- Threading behavior

**Test cases:**
- Webhook mode: messages go to webhook
- Bot mode: messages go to API with threading
- Disabled mode: no errors, no calls
- Thread timestamp propagation

**Approach:** Mock the Client interface.

---

### 10. `internal/metrics` - Tracker (Medium Complexity)

**File to create:** `internal/metrics/tracker_test.go`

**Functions to test:**
- `GetTodoCounts()` - Count aggregation
- `GetTodoItems()` - Item list
- `GetNewlyCompletedTodos()` - Delta detection
- `GetNewlyInProgressTodos()` - Delta detection
- `UpdatePreviousTodos()` - Snapshot update

**Test cases:**
- No previous todos, all new
- Some completed since last check
- Some started since last check
- No changes between checks
- Empty todo file

**Approach:** Create todo fixture files or mock file reading.

---

### 11. `cmd/ralph` - Runner (Integration, Most Complex)

**File to create:** `cmd/ralph/runner_test.go`

**Functions to test:**
- `Run()` - Main loop (with extensive mocking)
- `runCodeReviewPhase()` - Review loop
- `runCleanupPhase()` - File cleanup
- `runPRPhase()` - PR creation
- `generatePRTitle()` - Title generation
- `generatePRBody()` - Body generation
- `copyFile()` - File copy helper
- `cleanupWorktree()` - Cleanup helper
- `printSummary()` - Summary output

**Test cases:**
- Single iteration success
- Max iterations reached
- Max time reached
- User completion (exit code 0)
- Signal handling (SIGINT, SIGTERM)
- Worktree mode vs in-place mode
- Code review phase sequencing
- Cleanup phase with patterns
- PR phase success/failure
- PR title/body generation with various states

**Approach:** Create comprehensive mocks for Claude, git, Slack, metrics. Test in isolation first.

---

## Interfaces to Create

Create these interfaces to enable mocking:

### `internal/claude/interfaces.go`
```go
type Runner interface {
    Start() error
    Wait() (int, error)
    StreamOutput() <-chan string
    Kill() error
}
```

### `internal/git/interfaces.go`
```go
type CommandExecutor interface {
    Run(name string, args ...string) ([]byte, error)
}

type PRCreator interface {
    CreatePR(cfg PRConfig) (*PRResult, error)
}

type CommitTracker interface {
    CommitsDelta() (int, error)
    UpdateBaseline() error
}
```

### `internal/slack/interfaces.go`
```go
type Messenger interface {
    PostMessage(ctx context.Context, req *ChatPostMessageRequest) (*ChatPostMessageResponse, error)
    PostWebhook(ctx context.Context, msg *WebhookMessage) error
    IsConfigured() bool
}
```

### `internal/metrics/interfaces.go`
```go
type Collector interface {
    RecordIterationComplete(ctx context.Context, duration time.Duration, exitReason string)
    RecordCommits(ctx context.Context, count int)
    RecordError(ctx context.Context, errType string)
    UpdateTodoCounts(pending, completed int)
    Shutdown(ctx context.Context) error
}
```

---

## Test Fixtures

Create `testdata/` directories in relevant packages:

```
internal/todo/testdata/
├── empty.md
├── all_pending.md
├── all_completed.md
├── mixed.md
└── nested.md

internal/config/testdata/
├── minimal.yaml
├── full.yaml
├── invalid.yaml
└── partial.yaml

internal/claude/testdata/
├── user_message.json
├── assistant_message.json
├── todo_write.json
├── edit_result.json
└── system_reminder.json
```

---

## Implementation Order

1. **`internal/todo`** - Start here. Pure parsing, no dependencies.
2. **`internal/slack/messages`** - Pure formatting functions.
3. **`internal/config`** - File I/O but straightforward.
4. **`internal/claude/parser`** - JSON parsing, may need output capture.
5. **Create interfaces** - Before tackling components with external deps.
6. **`internal/git`** - With command executor mock.
7. **`internal/worktree`** - With command executor mock.
8. **`internal/slack/client`** - With HTTP mocking.
9. **`internal/metrics`** - With mocked dependencies.
10. **`internal/claude/client`** - With process mocking.
11. **`cmd/ralph/runner`** - Full integration with all mocks.

---

## Coverage Goals

| Package | Target Coverage |
|---------|-----------------|
| `internal/todo` | 90%+ |
| `internal/config` | 85%+ |
| `internal/claude` | 80%+ |
| `internal/git` | 75%+ |
| `internal/worktree` | 75%+ |
| `internal/slack` | 80%+ |
| `internal/metrics` | 75%+ |
| `cmd/ralph` | 70%+ |

---

## Testing Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific package
go test ./internal/todo/...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestParseFile ./internal/todo/...
```

---

## Notes

- Use `t.Parallel()` for tests that don't share state
- Use `t.Helper()` in test helper functions
- Prefer table-driven tests for multiple cases
- Use `testify/assert` or `testify/require` if desired (not currently a dependency)
- Keep test files in the same package for access to unexported functions
- Consider `go-cmp` for struct comparisons
