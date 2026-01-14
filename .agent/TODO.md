# TUI Issues TODO

- [x] Add `ParseAndPrintToTUI` to `internal/claude/parser.go`
- [x] Add `ParseAndPrintTo` to `internal/codex/parser.go`
- [x] Add `Model` field and `SetModel` to TUI state
- [x] Update renderer to display model alongside phase
- [x] Add `SetModel` method to TUI
- [x] Update runner to route output through TUI
- [x] Update runner to reset tokens per iteration
- [x] Update runner to call SetTodos at iteration start
- [x] Add `updateCurrentTodo` function to runner
- [x] Call `SetModel` at runner startup
- [x] Run tests to verify no regressions
