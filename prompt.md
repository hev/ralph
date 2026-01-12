# Plan: GitHub Issue as Prompt Input

Support starting a ralph loop from a GitHub issue number or URL. The issue becomes the prompt.md content.

## Overview

User provides a GitHub issue reference (number like `42` or URL like `https://github.com/owner/repo/issues/42`) and ralph fetches the issue, generates a prompt.md, and starts the loop.

## Input Formats

Accept these input patterns:
- Issue number: `42`, `#42`
- Full URL: `https://github.com/owner/repo/issues/42`
- Short reference: `owner/repo#42`

## Implementation

### 1. New Package: `internal/github/issue.go`

Create functions to:
- Parse issue references into owner/repo/number
- Fetch issue via `gh issue view <number> --json title,body,labels,assignees,milestone`
- Return structured `Issue` type with title, body, labels, etc.

```go
type Issue struct {
    Number    int
    Title     string
    Body      string
    Labels    []string
    URL       string
    Assignees []string
    Milestone string
}

func ParseIssueRef(ref string) (owner, repo string, number int, err error)
func FetchIssue(number int) (*Issue, error)
```

### 2. Prompt Generation: `internal/github/prompt.go`

Convert issue to prompt.md content:

```markdown
# Issue #42: <title>

<issue body>

---
Source: <issue URL>
Labels: <comma-separated labels>
```

Function signature:
```go
func GeneratePrompt(issue *Issue) string
```

### 3. CLI Changes: `cmd/ralph/main.go`

Add new flag:
- `-i, --issue STRING` - GitHub issue number or URL

Add logic in PreRunE or RunE:
- If `--issue` provided and no `-p` prompt file:
  - Parse issue reference
  - Fetch issue details
  - Generate prompt content
  - Write to `prompt.md` (or temp file)
  - Set prompt path to generated file

### 4. Config Changes: `internal/config/config.go`

Add to Config struct:
```go
Issue string `yaml:"issue"` // GitHub issue reference
```

Add to flag binding and YAML loading.

### 5. Worktree Integration

When `--issue` is used with `-w` worktree mode:
- Auto-generate branch name from issue: `ralph/issue-42-short-title`
- Use issue number in worktree path

Modify `internal/worktree/worktree.go`:
```go
func BranchNameFromIssue(issue *Issue) string
```

### 6. PR Linking

When creating PR with `--pr` after working on an issue:
- Include `Fixes #42` or `Closes #42` in PR body
- Reference the original issue URL

Modify `internal/runner/runner.go` in `runPRPhase`:
- Check if session started from issue
- Add issue reference to PR body template

### 7. Issue Comment (Optional Enhancement)

Post a comment on the issue when work starts/completes:
- `gh issue comment <number> --body "Started working on this issue..."`
- Include branch name, PR link when done

New functions in `internal/github/`:
```go
func CommentOnIssue(number int, body string) error
func LinkPRToIssue(issueNumber, prNumber int) error
```

## File Changes Summary

| File | Change |
|------|--------|
| `cmd/ralph/main.go` | Add `--issue` flag, issue fetching logic |
| `internal/config/config.go` | Add `Issue` field to Config |
| `internal/github/issue.go` | New - issue parsing and fetching |
| `internal/github/prompt.go` | New - prompt generation from issue |
| `internal/worktree/worktree.go` | Add `BranchNameFromIssue()` |
| `internal/runner/runner.go` | Pass issue context, modify PR body |
| `internal/git/pr.go` | Add issue reference to PR body |

## Usage Examples

```bash
# Start loop from issue number (uses current repo)
ralph -i 42

# Start loop from issue URL
ralph -i https://github.com/owner/repo/issues/42

# With worktree (auto-names branch from issue)
ralph -i 42 -w

# Full workflow: issue -> worktree -> PR
ralph -i 42 -w --pr

# With iteration limits
ralph -i 42 -n 10 -t 30m
```

## Testing

- Unit tests for issue reference parsing (various formats)
- Unit tests for prompt generation
- Integration test with mock `gh` output
- Add test scenario to `internal/testmode/` for issue-based runs

## Error Handling

- `gh` CLI not installed: clear error message with install link
- Issue not found: "Issue #42 not found in owner/repo"
- No repo context: "Cannot determine repository. Use full URL or run from a git repo"
- Rate limiting: surface GitHub API errors clearly

## Future Enhancements

- Support issue templates for prompt formatting
- Fetch issue comments for additional context
- Support GitLab/other forges via adapter pattern
- Auto-assign issue when starting work
- Update issue labels (e.g., add "in-progress")
