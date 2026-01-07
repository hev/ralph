# Ralph

Run Claude in a loop. Like Ralph Wiggum, the defaults are insane (unlimited iterations, unlimited time).

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
  -v, --version               Show version

OTEL Options:
  --otel-enabled              Enable metrics export (default: false)
  --otel-endpoint URL         OTLP endpoint (default: localhost:4317)
  --metrics-prefix PREFIX     Metric name prefix (default: ralph)
  --project-name NAME         Override project label (default: cwd basename)
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
```

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
│   ├── todo/parser.go        # TODO.md parsing
│   └── git/tracker.go        # Git commit counting
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
