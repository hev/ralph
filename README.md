# Ralph

Run Claude in a loop. Like Ralph Wiggum, the defaults are insane (unlimited iterations, unlimited time).

![source](https://github.com/user-attachments/assets/0d31df40-2edd-45c0-b57f-6b976fb453e2)

## Installation

```bash
# Build from source
make build

# Or install to GOPATH/bin
make install

# Or go install directly
go install github.com/hev/ralph/cmd/ralph@latest
```

## Usage

```bash
ralph [OPTIONS]
```

### Options

```
  -h, --help                  Show this help message
  -p, --prompt FILE           Path to prompt file (default: ./prompt.md)
  -n, --max-iterations N      Max loop iterations (default: 0 = unlimited)
  -t, --max-time SECONDS      Max total runtime in seconds (default: 0 = unlimited)
  -d, --agent-dir DIR         Scratchpad directory (default: ./.agent)
  -c, --cooldown SECONDS      Delay between iterations (default: 1)
  -q, --quiet                 Disable verbose output
  --dry-run                   Show what would run without executing
  --config FILE               Path to config file (default: ./ralph.yaml)
  -v, --version               Show version

OTEL Options:
  --otel-enabled              Enable metrics export (default: false)
  --otel-endpoint URL         OTLP endpoint (default: localhost:4317)
  --metrics-prefix PREFIX     Metric name prefix (default: ralph)
  --project-name NAME         Override project label (default: cwd basename)

Worktree Options:
  -w, --worktree              Run in a git worktree (default: false)
  -b, --branch NAME           Branch name for worktree (default: auto-generate)
  -k, --keep-worktree         Keep worktree after completion (default: false)
```

### Examples

```bash
# Run forever with defaults
ralph

# Run for 5 iterations
ralph -n 5

# Run for 1 hour
ralph -t 3600

# Use custom prompt file
ralph -p ~/tasks/build.md

# 10 iterations, 5s cooldown
ralph -n 10 -c 5

# With metrics enabled
ralph --otel-enabled --otel-endpoint localhost:4317

# Use custom config file
ralph --config ~/myconfig.yaml

# Run in a worktree (auto-generated branch)
ralph -w

# Run in worktree with specific branch
ralph -w -b feature/my-task

# Run in worktree, keep after completion
ralph -w -k
```

## Configuration File

Ralph supports a two-tier configuration system with global and local config files.

### Config File Locations

1. **Global config**: `~/.config/ralph/ralph.yaml` - Shared settings across all projects
2. **Local config**: `./ralph.yaml` - Project-specific overrides

Both files are loaded automatically (global first, then local). Local values override global values.

You can also specify a single config file with `--config` (bypasses two-tier loading).

### Example: Two-Tier Setup

**Global** (`~/.config/ralph/ralph.yaml`):
```yaml
# Shared across all projects
slack:
  enabled: true
  bot_token: xoxb-...  # Your bot token - shared across projects
```

**Local** (`./ralph.yaml`):
```yaml
# Project-specific settings
slack:
  channel: C0123456789        # This project's channel
  notify_users: U0123,U0456   # Users to notify for this project
```

### Full Configuration Reference

```yaml
# Core options
prompt: ./prompt.md
max_iterations: 10
max_time: 3600
agent_dir: ./.agent
cooldown: 5
verbose: true

# OpenTelemetry
otel:
  enabled: true
  endpoint: localhost:4317
  metrics_prefix: ralph
  project_name: my-project

# Slack notifications
slack:
  enabled: true
  webhook_url: https://hooks.slack.com/services/...
  bot_token: xoxb-...
  channel: C0123456789
  notify_users: U0123,U0456

# Custom scratchpad instructions (replaces the default)
scratchpad_prompt: "Use {{.AgentDir}} for notes. Track tasks in {{.AgentDir}}/TODO.md."
```

### Configuration Precedence

Configuration is loaded in this order (later sources override earlier):

1. Default values
2. Global config file (`~/.config/ralph/ralph.yaml`)
3. Local config file (`./ralph.yaml`)
4. Environment variables (for Slack options)
5. Command-line flags

## How It Works

Ralph runs Claude Code in a loop with `--dangerously-skip-permissions`. Each iteration:

1. Loads your prompt file
2. Appends scratchpad instructions (use `.agent/TODO.md` for tracking)
3. Runs Claude with streaming JSON output
4. Parses and displays colored output
5. Waits for cooldown period
6. Repeats until limits reached or interrupted

The scratchpad instructions tell Claude to:
- Use the agent directory as a scratchpad
- Track progress in `TODO.md` using checkboxes
- Make commits after each file edit
- Work on one task at a time

You can customize these instructions with the `scratchpad_prompt` config option. Use `{{.AgentDir}}` as a placeholder for the agent directory path.

## Observability

Ralph can export metrics to OpenTelemetry for monitoring in Grafana.

### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `ralph_iterations_total` | Counter | Total iterations completed |
| `ralph_iteration_duration_seconds` | Histogram | Time per iteration |
| `ralph_session_duration_seconds` | Gauge | Current session runtime |
| `ralph_commits_total` | Counter | Git commits made during session |
| `ralph_todos_pending` | Gauge | Current pending todo items |
| `ralph_todos_completed` | Gauge | Current completed todo items |
| `ralph_claude_errors_total` | Counter | Claude execution errors |
| `ralph_active_sessions` | Gauge | Currently running ralph instances |

All metrics include `project` and `session_id` labels.

### Start the Observability Stack

```bash
# Start OTEL collector, Prometheus, and Grafana
make up

# Run ralph with metrics
make run-otel

# View Grafana dashboard
open http://localhost:3000
# Login: admin/admin
```

### Stop the Stack

```bash
make down
```

## Slack Notifications

Ralph can send notifications to Slack when sessions start, todos are completed, and sessions end.

### Setup

There are two ways to configure Slack notifications:

#### Option 1: Webhook URL (Simple)

Use an [Incoming Webhook](https://api.slack.com/messaging/webhooks) for basic notifications. This method posts messages but doesn't support threaded replies.

1. Create a Slack app at https://api.slack.com/apps
2. Enable **Incoming Webhooks** and create a webhook for your channel
3. Copy the webhook URL

```bash
export RALPH_SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../xxx"
ralph --slack-enabled
```

#### Option 2: Bot Token (Recommended)

Use a bot token for full functionality including threaded replies for todo completions.

1. Create a Slack app at https://api.slack.com/apps
2. Under **OAuth & Permissions**, add these scopes:
   - `chat:write` - Send messages
   - `chat:write.public` - Post to channels without joining
3. Install the app to your workspace
4. Copy the **Bot User OAuth Token** (starts with `xoxb-`)
5. Get the channel ID (right-click channel > View channel details > copy ID at bottom)

```bash
export RALPH_SLACK_BOT_TOKEN="xoxb-..."
export RALPH_SLACK_CHANNEL="C0123456789"
ralph --slack-enabled
```

### Configuration Options

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--slack-enabled` | - | Enable Slack notifications |
| `--slack-webhook-url` | `RALPH_SLACK_WEBHOOK_URL` | Webhook URL for posting messages |
| `--slack-bot-token` | `RALPH_SLACK_BOT_TOKEN` | Bot token for API access |
| `--slack-channel` | `RALPH_SLACK_CHANNEL` | Channel ID (required with bot token) |
| `--slack-notify-users` | `RALPH_SLACK_NOTIFY_USERS` | Comma-separated user IDs to @mention on completion |

### Notification Types

**Session Start** - Posted when ralph begins:
- Project name and GitHub URL (if available)
- tmux session name (if running in tmux)
- Session ID and configured limits

**Todo Completed** (bot token only) - Threaded reply when a todo item is checked off:
- Todo text that was completed
- Progress (X/Y complete)
- Iteration number, commit count, duration

**Session End** - Summary when ralph finishes:
- Total iterations, duration, commits
- Todo completion percentage
- Exit reason (completed, max iterations, timeout, interrupted)
- @mentions configured users

### Examples

```bash
# Basic webhook setup
ralph --slack-enabled --slack-webhook-url "https://hooks.slack.com/services/..."

# Full bot setup with user notifications
ralph --slack-enabled \
  --slack-bot-token "xoxb-..." \
  --slack-channel "C0123456789" \
  --slack-notify-users "U0123456789,U9876543210"

# Using environment variables
export RALPH_SLACK_BOT_TOKEN="xoxb-..."
export RALPH_SLACK_CHANNEL="C0123456789"
export RALPH_SLACK_NOTIFY_USERS="U0123456789"
ralph --slack-enabled
```

## Worktree Mode

Ralph can optionally run in a git worktree for better isolation. This is opt-in; the default "insane" mode runs in the current directory.

### Why Worktrees?

- **Isolation**: Each Ralph session works in its own directory
- **Parallel sessions**: Multiple Ralphs can work on different branches simultaneously
- **Clean state**: Fresh worktree avoids contamination from previous work
- **Easy cleanup**: Just delete the worktree when done

### Usage

```bash
# Default: run in current directory (insane mode)
ralph

# Run in a worktree (auto-generated branch name)
ralph --worktree
ralph -w

# Specify branch name
ralph --worktree --branch feature/my-task
ralph -w -b feature/my-task

# Keep worktree after completion (skip cleanup)
ralph --worktree --keep-worktree
ralph -w -k
```

### How It Works

When worktree mode is enabled:

1. Creates a new git worktree at `/tmp/ralph-worktrees/<branch-name>`
2. Creates a new branch (or uses existing if specified)
3. Copies the prompt file to the worktree
4. Changes to the worktree directory
5. Runs Claude in the worktree context
6. On completion: pushes the branch to remote
7. Removes the worktree (unless `--keep-worktree`)

### Configuration

```yaml
# ralph.yaml
worktree:
  enabled: false          # Default: insane mode (no worktree)
  base_dir: /tmp/ralph-worktrees  # Where to create worktrees
  branch_prefix: ralph/   # Prefix for auto-generated branch names
  cleanup: true           # Delete worktree on completion
```

### CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--worktree` | `-w` | Enable worktree mode |
| `--branch NAME` | `-b` | Branch name for worktree (default: auto-generate) |
| `--keep-worktree` | `-k` | Keep worktree after completion |

## Project Structure

```
ralph/
├── cmd/ralph/main.go         # CLI entry point
├── internal/
│   ├── config/config.go      # Configuration
│   ├── runner/runner.go      # Main loop logic
│   ├── claude/
│   │   ├── client.go         # Claude process execution
│   │   └── parser.go         # JSON stream parser
│   ├── metrics/
│   │   ├── collector.go      # OTEL metrics setup
│   │   └── tracker.go        # Metric tracking helpers
│   ├── slack/
│   │   ├── client.go         # Slack API client
│   │   ├── messages.go       # Message formatting
│   │   └── notifier.go       # High-level notification logic
│   ├── todo/parser.go        # TODO.md parsing
│   ├── git/tracker.go        # Git commit counting
│   └── worktree/worktree.go  # Git worktree management
├── grafana/
│   └── ralph-dashboard.json  # Grafana dashboard
├── docker-compose.yml        # Observability stack
├── otel-collector-config.yaml
├── prometheus.yml
└── Makefile
```

## Development

```bash
# Build
make build

# Run tests
make test

# Clean
make clean

# Build for all platforms
make build-all
```

---

*"I'm in danger!"* - Ralph Wiggum
