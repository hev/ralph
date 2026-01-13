# Terminal UI Implementation - TODO

## Overview
Add a hybrid terminal UI with scrollable log buffer, pinned status footer, and keyboard shortcuts.

## Phase 1: Foundation
- [x] Add lipgloss dependency to go.mod
- [x] Create `internal/tui/` package directory
- [x] Create `internal/tui/styles.go` - lipgloss style definitions
- [x] Create `internal/tui/state.go` - UIState struct with thread-safe getters/setters

## Phase 2: Terminal Control
- [x] Create `internal/tui/terminal.go` - Terminal struct with TTY detection, EnterRawMode(), restore(), WatchResize()
- [x] Create `internal/tui/input.go` - InputReader for stdin raw mode, escape sequence parsing

## Phase 3: Buffer & Rendering
- [x] Create `internal/tui/buffer.go` - Ring buffer (10k default), Append(), GetVisibleLines(), ScrollUp/Down(), TogglePause()
- [x] Create `internal/tui/renderer.go` - ANSI cursor positioning, scroll region, drawFooter() with progress bars

## Phase 4: Main Orchestration
- [x] Create `internal/tui/tui.go` - New(), Start(), Stop() lifecycle; public API: WriteLine(), SetPhase(), SetTodos(), SetTokens()

## Phase 5: Parser Refactor
- [x] Modify `internal/claude/parser.go` - Add ParseOutput() that returns structured data (ParsedOutput{Content, LineType, Tokens})
- [x] Keep ParseAndPrint() for backwards compatibility but have it call ParseOutput internally

## Phase 6: Runner Integration
- [x] Add `TUIBufferSize` to Config struct with default 10000
- [x] Add `tui_buffer_size` to yamlConfig struct and LoadFromFile()
- [x] Modify runner.go - Initialize TUI at start of Run()
- [x] Modify runner.go - Replace log() calls with ui.WriteLine()
- [x] Modify runner.go - Call ui.SetPhase() at phase transitions
- [x] Modify runner.go - Call ui.SetTodos() after each iteration
- [x] Modify runner.go - Track cumulative tokens and call ui.SetTokens()

## Phase 7: Non-TTY Fallback
- [x] Add IsTerminal check in TUI initialization
- [x] Implement passthrough mode - WriteLine writes directly to stdout
- [x] Skip raw mode, render loop, input handling when not a TTY
- [x] Add --no-tui flag to disable terminal UI for debugging

## Phase 8: Testing & Verification
- [x] Run existing tests to ensure no regressions
- [ ] Add TUI unit tests for buffer, state, and rendering
- [ ] Manual test: basic rendering with simple prompt
- [ ] Manual test: keyboard controls (space, arrows, j/k, g/G)
- [ ] Manual test: phase transitions with --code-review flag
- [ ] Manual test: resize handling
- [ ] Manual test: non-TTY piping (`ralph ... > out.txt`)
- [ ] Manual test: clean shutdown with Ctrl+C

## Footer Layout Reference
```
┌─────────────────────────────────────────────────────┐
│ [logs scroll here...]                               │
├─────────────────────────────────────────────────────┤
│  Main Loop  [PAUSED]                                │  <- Phase + pause indicator
│  Todos: ████████░░░░░░░░ 5/12 (42%)                │  <- Todo progress bar
│  Working on: Fix authentication bug                 │  <- Current in-progress todo
│  Context: ██████░░░░░░░░░ 45k/200k                  │  <- Token utilization
└─────────────────────────────────────────────────────┘
```

## Keyboard Controls Reference
| Key | Action |
|-----|--------|
| Space | Toggle pause/resume auto-scroll |
| ↑/k | Scroll up one line |
| ↓/j | Scroll down one line |
| Page Up | Scroll up one page |
| Page Down | Scroll down one page |
| g | Jump to top |
| G | Jump to bottom |
