# TUI Issues TODO

- [x] Add `ParseAndPrintToTUI` to `internal/claude/parser.go`
- [x] Add `ParseAndPrintTo` to `internal/codex/parser.go`
- [ ] Add `Model` field and `SetModel` to TUI state
- [ ] Update renderer to display model alongside phase
- [ ] Add `SetModel` method to TUI
- [ ] Update runner to route output through TUI
- [ ] Update runner to reset tokens per iteration
- [ ] Update runner to call SetTodos at iteration start
- [ ] Add `updateCurrentTodo` function to runner
- [ ] Call `SetModel` at runner startup
- [ ] Run tests to verify no regressions
