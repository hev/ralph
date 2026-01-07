# Ralph - Go Rewrite with Observability

Convert the `ralph` bash script to a Go binary while preserving all current functionality and adding OpenTelemetry-based metrics for Grafana dashboard monitoring.

## Current Feature Set (Preserve All)

### CLI Options
- `-p, --prompt FILE` - Path to prompt file (default: ./prompt.md)
- `-n, --max-iterations N` - Max loop iterations (0 = unlimited)
- `-t, --max-time SECONDS` - Max total runtime (0 = unlimited)
- `-d, --agent-dir DIR` - Scratchpad directory (default: ./.agent)
- `-c, --cooldown SECONDS` - Delay between iterations (default: 1)
- `-q, --quiet` - Disable verbose output
- `--dry-run` - Show what would run without executing
- `-h, --help` - Show help
- `-v, --version` - Show version

### Core Behavior
1. Load and validate prompt file
2. Create agent directory if missing
3. Append scratchpad instructions to prompt
4. Run `claude --dangerously-skip-permissions --print --verbose --output-format stream-json -p "$PROMPT"` in a loop
5. Parse streaming JSON output and format with colors
6. Handle iteration/time limits
7. Graceful shutdown on SIGINT/SIGTERM
8. Print summary on exit (iterations, time, exit reason)

### Output Formatting
- Parse JSON stream for message types: `user`, `assistant`, `result`, `system`
- Color-coded output (blue, green, yellow, red)
- Tool call display with name and truncated input
- Result truncation for long outputs

---

## New Observability Components

### 1. OpenTelemetry Metrics

Add OTEL metrics exporter that pushes to a collector. Use the OTLP exporter.

**New CLI Flags:**
- `--otel-endpoint URL` - OTLP endpoint (default: localhost:4317)
- `--otel-enabled` - Enable metrics export (default: false)
- `--metrics-prefix` - Metric name prefix (default: ralph)
- `--project-name NAME` - Override project label (default: cwd basename)

**Metrics to Track:**

All metrics include `project` label (working directory basename or full path).

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `ralph_iterations_total` | Counter | `project`, `session_id`, `exit_reason` | Total iterations completed |
| `ralph_iteration_duration_seconds` | Histogram | `project`, `session_id` | Time per iteration |
| `ralph_session_duration_seconds` | Gauge | `project`, `session_id` | Current session runtime |
| `ralph_commits_total` | Counter | `project`, `session_id` | Git commits made during session |
| `ralph_todos_pending` | Gauge | `project`, `session_id` | Current pending todo items |
| `ralph_todos_completed` | Gauge | `project`, `session_id` | Current completed todo items |
| `ralph_claude_errors_total` | Counter | `project`, `session_id`, `error_type` | Claude execution errors |
| `ralph_active_sessions` | Gauge | `project` | Currently running ralph instances |

**Project Label:**
- Derived from current working directory at startup
- Use directory basename by default (e.g., `my-app`)
- Optional `--project-name` flag to override

### 2. Todo Tracking

Parse `${AGENT_DIR}/TODO.md` after each iteration to extract todo counts:
- Count lines matching `- [ ]` as pending
- Count lines matching `- [x]` as completed
- Update gauges after each iteration

### 3. Commit Tracking

Track git commits by either:
- Option A: Parse git log before/after each iteration to detect new commits
- Option B: Count commits with author matching claude's pattern
- Option C: Hook into git output from claude's stream

Recommend Option A: `git rev-list --count HEAD` before and after iteration.

### 4. Session Management

Generate unique session ID at startup (UUID or timestamp-based) for metric labels. This enables filtering dashboards by session.

---

## Grafana Dashboard

Create a JSON dashboard definition (`grafana/ralph-dashboard.json`) with:

### Panels

1. **Sessions Overview** (Stat)
   - Active sessions count
   - Total iterations (all sessions)

2. **Iterations Over Time** (Time series)
   - `rate(ralph_iterations_total[5m])` by session
   - Shows iteration velocity

3. **Iteration Duration** (Heatmap)
   - Distribution of iteration durations
   - Identify slow iterations

4. **Todo Progress** (Gauge + Time series)
   - Current pending vs completed ratio
   - Todo completion rate over time

5. **Commits Over Time** (Time series)
   - `rate(ralph_commits_total[5m])`
   - Commit velocity per session

6. **Error Rate** (Time series)
   - Claude errors by type
   - Alert threshold indicator

7. **Session Summary Table**
   - Project, session ID, start time, iterations, commits, todos done
   - Filterable/sortable by project

### Variables
- `project` - Dropdown to filter by project
- `session_id` - Dropdown to filter by session (filtered by selected project)
- `time_range` - Standard Grafana time picker

---

## Project Structure

```
ralph/
├── cmd/
│   └── ralph/
│       └── main.go           # Entry point, CLI parsing
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration struct and loading
│   ├── runner/
│   │   └── runner.go         # Main loop logic
│   ├── claude/
│   │   ├── client.go         # Claude process execution
│   │   └── parser.go         # JSON stream parser
│   ├── metrics/
│   │   ├── collector.go      # OTEL metrics setup
│   │   └── tracker.go        # Metric tracking helpers
│   ├── todo/
│   │   └── parser.go         # TODO.md parsing
│   └── git/
│       └── tracker.go        # Git commit counting
├── grafana/
│   └── ralph-dashboard.json  # Dashboard definition
├── docker-compose.yml        # OTEL collector + Grafana stack
├── otel-collector-config.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Implementation Order

1. **Core CLI** - cobra/viper setup, config parsing, help/version
2. **Runner Loop** - Main iteration logic without metrics
3. **Claude Client** - Process execution with streaming JSON
4. **JSON Parser** - Stream parsing and colored output
5. **Signal Handling** - Graceful shutdown
6. **Todo Parser** - Parse TODO.md for counts
7. **Git Tracker** - Commit counting
8. **OTEL Metrics** - Metric collector and exporters
9. **Dashboard** - Grafana JSON definition
10. **Docker Stack** - Compose file for local observability

---

## Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/fatih/color v1.16.0
    github.com/google/uuid v1.5.0
    go.opentelemetry.io/otel v1.22.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.45.0
    go.opentelemetry.io/otel/metric v1.22.0
    go.opentelemetry.io/otel/sdk/metric v1.22.0
)
```

---

## Docker Compose Stack

```yaml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
    volumes:
      - ./otel-collector-config.yaml:/etc/otelcol-contrib/config.yaml

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ./grafana/ralph-dashboard.json:/var/lib/grafana/dashboards/ralph.json
```

---

## Acceptance Criteria

- [ ] All existing CLI flags work identically to bash version
- [ ] Colored output matches current format
- [ ] Graceful shutdown preserves summary output
- [ ] Metrics export to OTEL collector when enabled
- [ ] Todo counts update after each iteration
- [ ] Commit counts tracked accurately
- [ ] Grafana dashboard shows all defined panels
- [ ] Docker compose brings up full observability stack
- [ ] `ralph --help` output matches current style
- [ ] Binary runs without Docker for users who don't need metrics

---

## Slack Notification Hooks

Add Slack integration to notify teams of ralph session progress via threaded messages.

### Behavior Overview

1. **Session Start**: Create a new Slack thread when ralph starts
2. **Todo Completion**: Update the thread each time a todo item transitions to completed
3. **Session End**: Post final summary and @mention configured users

### New CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--slack-enabled` | bool | false | Enable Slack notifications |
| `--slack-webhook-url` | string | "" | Slack webhook URL (or `RALPH_SLACK_WEBHOOK_URL` env var) |
| `--slack-channel` | string | "" | Channel ID to post to |
| `--slack-notify-users` | string | "" | Comma-separated Slack user IDs to @mention on completion |
| `--slack-bot-token` | string | "" | Bot token for thread replies (or `RALPH_SLACK_BOT_TOKEN` env var) |

### Thread Starter Message (Session Start)

Posted when ralph begins. Contains:

```
🤖 Ralph session started

📁 Project: my-project
🔗 GitHub: https://github.com/user/my-project
🖥️ tmux: session-name (if $TMUX is set)
⏱️ Started: 2025-01-07 14:30:00 UTC
🆔 Session: abc123-def456

Limits: 10 iterations / 3600s max
```

**Implementation notes:**
- Extract GitHub URL from `git remote get-url origin`
- Parse tmux session from `$TMUX` env var (format: `/tmp/tmux-1000/default,12345,0` → extract socket path, get session name via `tmux display-message -p '#S'`)
- Use Slack webhook for initial post, capture `ts` (timestamp) for thread replies
- If tmux not available, omit that line

### Thread Updates (Todo Completion)

Posted as replies to the thread when a todo item is marked complete:

```
✅ Todo completed (3/7)
"Implement user authentication"

Iteration: 5 | Commits: 2 | Duration: 45s
```

**Trigger logic:**
- Compare todo counts before/after each iteration (already tracked in `metrics.Tracker`)
- If `completed` count increased, post update for each newly completed item
- Parse TODO.md to get the actual todo text that was completed

### Final Message (Session End)

Posted when ralph exits (normally or via signal):

```
🏁 Ralph session complete

📊 Summary:
• Iterations: 15
• Duration: 23m 45s
• Commits: 8
• Todos: 7/7 complete (100%)

Exit reason: max iterations reached

cc: @user1 @user2
```

**Implementation notes:**
- @mentions use Slack user ID format: `<@U1234567890>`
- Only @mention if `--slack-notify-users` is configured
- Include exit reason from runner

### Project Structure Addition

```
internal/
├── slack/
│   ├── client.go      # Slack API client (webhook + bot token)
│   ├── notifier.go    # High-level notification logic
│   └── messages.go    # Message formatting/templates
```

### Config Additions

```go
// In config/config.go
type Config struct {
    // ... existing fields ...

    // Slack options
    SlackEnabled     bool
    SlackWebhookURL  string
    SlackChannel     string
    SlackNotifyUsers []string  // Parsed from comma-separated flag
    SlackBotToken    string
}
```

### Notifier Interface

```go
// internal/slack/notifier.go

type Notifier struct {
    client       *Client
    threadTS     string  // Thread timestamp for replies
    channel      string
    notifyUsers  []string
    projectName  string
    githubURL    string
    tmuxSession  string
    sessionID    string
}

// Called at session start
func (n *Notifier) SessionStart(ctx context.Context) error

// Called when todo item(s) complete
func (n *Notifier) TodoCompleted(ctx context.Context, todoText string, completed, total int, iteration int, commits int, iterDuration time.Duration) error

// Called at session end
func (n *Notifier) SessionEnd(ctx context.Context, summary SessionSummary) error

type SessionSummary struct {
    Iterations   int
    Duration     time.Duration
    Commits      int
    TodosDone    int
    TodosTotal   int
    ExitReason   string
}
```

### Integration Points

1. **runner.go:Run()** - Initialize notifier, call `SessionStart()` before loop, `SessionEnd()` in defer
2. **runner.go main loop** - After `tracker.AfterIteration()`, check for todo completions and call `TodoCompleted()`
3. **metrics/tracker.go** - Expose method to get previous vs current todo state for diff detection

### Todo Diff Detection

To detect which todos completed, need to track state across iterations:

```go
// In metrics/tracker.go or new todo/tracker.go

type TodoTracker struct {
    previousItems []TodoItem  // State from last iteration
    currentItems  []TodoItem  // Current state
}

type TodoItem struct {
    Text      string
    Completed bool
}

// Returns items that transitioned from pending → completed
func (t *TodoTracker) GetNewlyCompleted() []TodoItem
```

Parse TODO.md to extract full todo text (not just counts), compare between iterations.

### Environment Variables

Support env vars for sensitive values:
- `RALPH_SLACK_WEBHOOK_URL` - Webhook URL
- `RALPH_SLACK_BOT_TOKEN` - Bot OAuth token
- `RALPH_SLACK_CHANNEL` - Default channel
- `RALPH_SLACK_NOTIFY_USERS` - Default users to notify

CLI flags override env vars.

### Error Handling

- Slack failures should NOT stop ralph execution
- Log warnings on Slack errors, continue processing
- Retry with exponential backoff (1s, 2s, 4s) up to 3 attempts
- If initial thread creation fails, disable further Slack notifications for session

### Dependencies

```go
require (
    github.com/slack-go/slack v0.12.0  // Official Slack SDK
)
```

### Acceptance Criteria

- [ ] Thread created on session start with project info
- [ ] GitHub URL correctly extracted from git remote
- [ ] tmux session displayed when running in tmux
- [ ] Thread updated when todo items complete
- [ ] Completed todo text shown in update message
- [ ] Final summary posted on session end
- [ ] Configured users @mentioned in final message
- [ ] Slack failures don't crash ralph
- [ ] Sensitive tokens read from env vars
- [ ] Works with `--slack-enabled=false` (no-op)

---

## Stop on Completion

Add opt-in support for stopping the ralph loop when all todos are complete. Because this is ralph (insane defaults), the feature is disabled by default.

### Behavior

When `--stop-on-completion` is enabled:
1. After each iteration, check if TODO.md has todos and all are marked complete
2. If `Pending == 0 && Completed > 0`, exit the loop gracefully
3. Use exit reason: "all todos complete"
4. If no TODO.md exists or it has no items, continue running (don't stop on empty)

### New CLI Flag

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--stop-on-completion` | bool | false | Exit when all todos are complete |

### Config Additions

```go
// In config/config.go - Config struct
StopOnCompletion bool

// In yamlConfig struct
StopOnCompletion *bool `yaml:"stop_on_completion"`
```

### YAML Config

```yaml
# ralph.yaml
stop_on_completion: true
```

### Implementation Changes

#### 1. config/config.go

Add `StopOnCompletion` field to Config struct (default: false)

Add to `yamlConfig` struct:
```go
StopOnCompletion *bool `yaml:"stop_on_completion"`
```

Add to `LoadFromFile`:
```go
if yc.StopOnCompletion != nil {
    c.StopOnCompletion = *yc.StopOnCompletion
}
```

#### 2. cmd/ralph/main.go

Add CLI flag:
```go
rootCmd.Flags().BoolVar(&cfg.StopOnCompletion, "stop-on-completion", cfg.StopOnCompletion, "Exit when all todos are complete")
```

Add to `savedValues` handling in PreRunE:
```go
case "stop-on-completion":
    savedValues["stop-on-completion"] = cfg.StopOnCompletion
```

Add to value restoration:
```go
case "stop-on-completion":
    cfg.StopOnCompletion = val.(bool)
```

Update help template to include the new flag under a new section or after core options.

#### 3. runner/runner.go

Add completion check after the existing todo notification logic (around line 232):

```go
// Check for stop-on-completion
if cfg.StopOnCompletion {
    if counts, err := tracker.GetTodoCounts(); err == nil {
        if counts.Pending == 0 && counts.Completed > 0 {
            exitReason = "all todos complete"
            log("All todos complete, stopping...")
            break
        }
    }
}
```

### Exit Reason

The new exit reason "all todos complete" will be:
- Displayed in the summary output
- Sent to Slack if notifications enabled
- Recorded in OTEL metrics if enabled

### Files to Modify

1. `internal/config/config.go` - Add StopOnCompletion to Config and yamlConfig, update LoadFromFile
2. `cmd/ralph/main.go` - Add --stop-on-completion flag, update PreRunE handlers, update help template
3. `internal/runner/runner.go` - Add completion check in main loop after todo processing

### Acceptance Criteria

- [ ] `--stop-on-completion` flag available in CLI
- [ ] `stop_on_completion` available in YAML config
- [ ] Disabled by default (ralph philosophy)
- [ ] Stops loop when all todos are complete (Pending=0, Completed>0)
- [ ] Does NOT stop when TODO.md is empty or missing
- [ ] Exit reason "all todos complete" shown in summary
- [ ] Exit reason sent to Slack if enabled
- [ ] CLI flag overrides YAML config value

---

## Logging Improvements

Improve ralph's log output formatting to be more readable and reduce noise. All changes are in `internal/claude/parser.go`.

### 1. Format TodoWrite Tool Calls

When a `TodoWrite` tool is called, format the todo list as a readable checklist instead of raw JSON.

**Current output:**
```
[TOOL: TodoWrite]
{"todos":[{"activeForm":"Adding Context Window gauge panel","content":"Add Context Window gauge panel","status":"completed"},{"activeForm":"Adding User Input counter panel","content":"Add User Input counter panel","status":"in_progress"},...]}
```

**Desired output:**
```
[TOOL: TodoWrite]
  ✓ Add Context Window gauge panel
  ▶ Add User Input counter panel
  ○ Add Session links panel
  ○ Improve Cost/Model breakdown
```

**Implementation:**
- In `printAssistantMessage`, check if `block.Name == "TodoWrite"`
- Parse `block.Input` as `map[string]interface{}` to extract `todos` array
- Format each todo with status icons:
  - `completed` → `✓` (green)
  - `in_progress` → `▶` (yellow)
  - `pending` → `○` (dim)
- Print `content` field for each todo item

### 2. Strip System Reminder Tags

Filter out `<system-reminder>...</system-reminder>` blocks from all output.

**Implementation:**
- Create helper function `stripSystemReminders(text string) string`
- Use regex: `(?s)<system-reminder>.*?</system-reminder>`
- Apply to:
  - `block.Text` in user messages
  - `block.Content` and `block.Output` in tool results
  - `msg.Result` in result messages
- Strip the tags before printing, don't output anything about them

### 3. Truncate Read Tool Results

Reduce log bloat by truncating large file read outputs.

**Current:** Full file contents displayed
**Desired:** Summary with truncation

**Implementation:**
- In `printUserMessage`, when handling `tool_result` blocks:
- If result exceeds threshold (e.g., 500 chars), truncate with line count
- Format: `[Read: 245 lines] first few lines...`
- Show first 3-5 lines of content as preview

### 4. Special Formatting for .agents/TODO.md

When the Read tool reads `.agents/TODO.md` (or `${AGENT_DIR}/TODO.md`), format it as a checklist.

**Implementation:**
- Detect file path in tool result (may need to track from tool_use block)
- Parse markdown checklist format:
  - `- [ ]` → pending (○)
  - `- [x]` → completed (✓)
- Display as formatted checklist similar to TodoWrite

**Challenge:** Tool results don't include the file path. Options:
- Track last `Read` tool call's `file_path` parameter
- Detect by content pattern (starts with checklist items)
- Add state to parser to correlate tool_use → tool_result

### 5. Color-Coded Edit Formatting

Format Edit tool calls with diff-style coloring like Claude's output.

**Current output:**
```
[TOOL: Edit]
{"file_path":"/path/file.json","new_string":"new content","old_string":"old content","replace_all":false}
```

**Desired output:**
```
[TOOL: Edit] /path/file.json
  - old content
  + new content
```

**Implementation:**
- In `printAssistantMessage`, check if `block.Name == "Edit"`
- Parse `block.Input` to extract `file_path`, `old_string`, `new_string`
- Print file path on the header line
- Print `old_string` lines prefixed with `-` in red
- Print `new_string` lines prefixed with `+` in green
- Handle multi-line strings by prefixing each line
- Truncate if diff is very long (>20 lines each side)

### Files to Modify

1. `internal/claude/parser.go`:
   - Add `stripSystemReminders()` helper
   - Add `formatTodoWrite()` for TodoWrite tool formatting
   - Add `formatEdit()` for Edit tool formatting
   - Add `formatReadResult()` for truncating read results
   - Update `printAssistantMessage()` to use special formatters
   - Update `printUserMessage()` to strip reminders and truncate results

### New Helper Functions

```go
// Strip <system-reminder> tags from text
func stripSystemReminders(text string) string

// Format TodoWrite tool input as checklist
func formatTodoWrite(input interface{})

// Format Edit tool input as colored diff
func formatEdit(input interface{})

// Format/truncate Read tool result
func formatReadResult(content string, maxLines int) string

// Detect if content looks like a TODO.md checklist
func isTodoChecklist(content string) bool

// Format checklist content
func formatChecklist(content string)
```

### Acceptance Criteria

- [ ] TodoWrite calls display as formatted checklist with status icons
- [ ] System reminder tags are completely stripped from output
- [ ] Large Read results are truncated with line count summary
- [ ] TODO.md content displays as formatted checklist
- [ ] Edit calls display as colored diff (red removed, green added)
- [ ] File path shown in Edit tool header
- [ ] Multi-line edits handled correctly
- [ ] No raw JSON blobs in tool output for these tools
- [ ] Other tools continue to display as before
