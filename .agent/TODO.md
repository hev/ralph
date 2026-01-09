# Ralph Autonomy & UX Improvements

## Tasks

- [x] Improvement 1: Slack Progress Notifications - Post to Slack when Ralph starts working on a TODO item
- [x] Improvement 2: Automated Code Review Loop - Trigger code review when all TODOs complete
- [x] Improvement 3: Cleanup Step Post-Review - Remove test files and artifacts after review
- [x] Improvement 4: Model Selection Options - Add CLI and config options for model selection
- [x] Improvement 5: Automatic PR Creation - Add option to create PR when loop completes
- [x] Improvement 6: UX Review - Review help output and identify additional improvements

## Improvement 6 Findings (UX Review)

### Issues Found
1. **Missing `-v/--version` in help** - Flag exists but not shown in help output
2. **README outdated** - Doesn't document model selection, code review, cleanup, PR, stop-on-completion
3. **No `--cleanup-prompt` flag** - Inconsistent with code-review having one
4. **No cleanup patterns CLI flag** - Only configurable via YAML
5. **No `ralph init` command** - No easy way to generate starter config
6. **No `ralph status` command** - Can't check session state or config
7. **Cooldown default mismatch** - Help says 1, code defaults to 0
8. **No model validation** - Invalid model names silently fail
9. **No composite workflow flags** - Common patterns need many flags
10. **No progress bar/ETA** - No sense of progress in long sessions

### Recommended New Features
- `ralph init` - Generate starter config file
- `ralph status` - Show running sessions and config
- `ralph worktrees` - Manage worktrees (list, clean)
- `--profile` flag - Named configuration presets
- Progress display for long sessions

See `.agent/UX_REVIEW.md` for full details and prioritized recommendations.
