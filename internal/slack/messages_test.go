package slack

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "zero seconds",
			duration: 0,
			want:     "0s",
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			want:     "30s",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			want:     "59s",
		},
		{
			name:     "exactly 1 minute",
			duration: 1 * time.Minute,
			want:     "1m 0s",
		},
		{
			name:     "90 seconds",
			duration: 90 * time.Second,
			want:     "1m 30s",
		},
		{
			name:     "59 minutes 59 seconds",
			duration: 59*time.Minute + 59*time.Second,
			want:     "59m 59s",
		},
		{
			name:     "exactly 1 hour",
			duration: 1 * time.Hour,
			want:     "1h 0m",
		},
		{
			name:     "1 hour 30 minutes",
			duration: 1*time.Hour + 30*time.Minute,
			want:     "1h 30m",
		},
		{
			name:     "2 hours",
			duration: 2 * time.Hour,
			want:     "2h 0m",
		},
		{
			name:     "2 hours 45 minutes",
			duration: 2*time.Hour + 45*time.Minute,
			want:     "2h 45m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestTruncateSessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "full UUID",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			want: "550e8400-e29",
		},
		{
			name: "exactly 12 chars",
			id:   "123456789012",
			want: "123456789012",
		},
		{
			name: "short ID",
			id:   "abc123",
			want: "abc123",
		},
		{
			name: "empty string",
			id:   "",
			want: "",
		},
		{
			name: "13 chars",
			id:   "1234567890123",
			want: "123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateSessionID(tt.id)
			if got != tt.want {
				t.Errorf("truncateSessionID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestFormatSessionStart(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name           string
		info           SessionStartInfo
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "full info with all fields",
			info: SessionStartInfo{
				ProjectName:   "my-project",
				GitHubURL:     "https://github.com/user/repo",
				TmuxSession:   "ralph-session",
				StartTime:     startTime,
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				MaxIterations: 10,
				MaxTime:       3600,
			},
			wantContains: []string{
				"*Ralph session started*",
				"*Project:* my-project",
				"<https://github.com/user/repo|https://github.com/user/repo>",
				"*tmux:* `ralph-session`",
				"2024-01-15 10:30:00 UTC",
				"`550e8400-e29`",
				"10 iterations",
				"3600s max",
			},
		},
		{
			name: "minimal info - no GitHub URL or tmux",
			info: SessionStartInfo{
				ProjectName:   "minimal-project",
				StartTime:     startTime,
				SessionID:     "abc123",
				MaxIterations: 0,
				MaxTime:       0,
			},
			wantContains: []string{
				"*Ralph session started*",
				"*Project:* minimal-project",
				"unlimited iterations",
				"unlimited time",
			},
			wantNotContain: []string{
				"*GitHub:*",
				"*tmux:*",
			},
		},
		{
			name: "unlimited iterations with time limit",
			info: SessionStartInfo{
				ProjectName:   "test-project",
				StartTime:     startTime,
				SessionID:     "test-id",
				MaxIterations: 0,
				MaxTime:       7200,
			},
			wantContains: []string{
				"unlimited iterations",
				"7200s max",
			},
		},
		{
			name: "limited iterations with unlimited time",
			info: SessionStartInfo{
				ProjectName:   "test-project",
				StartTime:     startTime,
				SessionID:     "test-id",
				MaxIterations: 5,
				MaxTime:       0,
			},
			wantContains: []string{
				"5 iterations",
				"unlimited time",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatSessionStart(tt.info)

			if msg == nil {
				t.Fatal("FormatSessionStart returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(msg.Text, notWant) {
					t.Errorf("message should not contain %q, got:\n%s", notWant, msg.Text)
				}
			}
		})
	}
}

func TestFormatTodoStarted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		info         TodoStartedInfo
		wantContains []string
	}{
		{
			name: "first todo of many",
			info: TodoStartedInfo{
				TodoText:       "Implement feature X",
				CurrentIndex:   1,
				TotalCount:     5,
				Iteration:      1,
				CompletedCount: 0,
			},
			wantContains: []string{
				"*Working on item 1 of 5*",
				"_Implement feature X_",
				"Iteration: 1",
				"Completed: 0/5",
			},
		},
		{
			name: "middle todo with some completed",
			info: TodoStartedInfo{
				TodoText:       "Fix bug Y",
				CurrentIndex:   3,
				TotalCount:     10,
				Iteration:      5,
				CompletedCount: 4,
			},
			wantContains: []string{
				"*Working on item 3 of 10*",
				"_Fix bug Y_",
				"Iteration: 5",
				"Completed: 4/10",
			},
		},
		{
			name: "last todo",
			info: TodoStartedInfo{
				TodoText:       "Final task",
				CurrentIndex:   3,
				TotalCount:     3,
				Iteration:      10,
				CompletedCount: 2,
			},
			wantContains: []string{
				"*Working on item 3 of 3*",
				"_Final task_",
				"Iteration: 10",
				"Completed: 2/3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatTodoStarted(tt.info)

			if msg == nil {
				t.Fatal("FormatTodoStarted returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}
		})
	}
}

func TestFormatTodoCompleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		info         TodoCompletedInfo
		wantContains []string
	}{
		{
			name: "first todo completed",
			info: TodoCompletedInfo{
				TodoText:          "Implement feature X",
				CompletedCount:    1,
				TotalCount:        5,
				Iteration:         1,
				Commits:           2,
				IterationDuration: 45 * time.Second,
			},
			wantContains: []string{
				"*Todo completed (1/5)*",
				"_Implement feature X_",
				"Iteration: 1",
				"Commits: 2",
				"Duration: 45s",
			},
		},
		{
			name: "all todos completed",
			info: TodoCompletedInfo{
				TodoText:          "Final cleanup",
				CompletedCount:    10,
				TotalCount:        10,
				Iteration:         15,
				Commits:           25,
				IterationDuration: 2*time.Minute + 30*time.Second,
			},
			wantContains: []string{
				"*Todo completed (10/10)*",
				"_Final cleanup_",
				"Iteration: 15",
				"Commits: 25",
				"Duration: 2m 30s",
			},
		},
		{
			name: "zero commits",
			info: TodoCompletedInfo{
				TodoText:          "Documentation update",
				CompletedCount:    3,
				TotalCount:        5,
				Iteration:         4,
				Commits:           0,
				IterationDuration: 10 * time.Second,
			},
			wantContains: []string{
				"*Todo completed (3/5)*",
				"Commits: 0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatTodoCompleted(tt.info)

			if msg == nil {
				t.Fatal("FormatTodoCompleted returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}
		})
	}
}

func TestFormatSessionEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		summary        SessionSummary
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "successful completion with todos",
			summary: SessionSummary{
				Iterations:  10,
				Duration:    1*time.Hour + 30*time.Minute,
				Commits:     25,
				TodosDone:   8,
				TodosTotal:  10,
				ExitReason:  "All todos completed",
				NotifyUsers: []string{"U123ABC", "U456DEF"},
			},
			wantContains: []string{
				"*Ralph session complete*",
				"*Summary:*",
				"Iterations: 10",
				"Duration: 1h 30m",
				"Commits: 25",
				"Todos: 8/10 complete (80%)",
				"*Exit reason:* All todos completed",
				"<@U123ABC>",
				"<@U456DEF>",
				"cc:",
			},
		},
		{
			name: "max iterations reached - no todos",
			summary: SessionSummary{
				Iterations: 5,
				Duration:   30 * time.Minute,
				Commits:    10,
				TodosDone:  0,
				TodosTotal: 0,
				ExitReason: "Max iterations reached",
			},
			wantContains: []string{
				"*Ralph session complete*",
				"Iterations: 5",
				"Duration: 30m 0s",
				"Commits: 10",
				"*Exit reason:* Max iterations reached",
			},
			wantNotContain: []string{
				"Todos:",
				"cc:",
			},
		},
		{
			name: "100% completion rate",
			summary: SessionSummary{
				Iterations: 3,
				Duration:   15 * time.Minute,
				Commits:    5,
				TodosDone:  5,
				TodosTotal: 5,
				ExitReason: "User requested stop",
			},
			wantContains: []string{
				"Todos: 5/5 complete (100%)",
			},
		},
		{
			name: "0% completion rate",
			summary: SessionSummary{
				Iterations: 1,
				Duration:   5 * time.Minute,
				Commits:    0,
				TodosDone:  0,
				TodosTotal: 10,
				ExitReason: "Error encountered",
			},
			wantContains: []string{
				"Todos: 0/10 complete (0%)",
			},
		},
		{
			name: "single notify user",
			summary: SessionSummary{
				Iterations:  1,
				Duration:    1 * time.Minute,
				Commits:     1,
				TodosDone:   1,
				TodosTotal:  1,
				ExitReason:  "Complete",
				NotifyUsers: []string{"USINGLEX"},
			},
			wantContains: []string{
				"cc: <@USINGLEX>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatSessionEnd(tt.summary)

			if msg == nil {
				t.Fatal("FormatSessionEnd returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(msg.Text, notWant) {
					t.Errorf("message should not contain %q, got:\n%s", notWant, msg.Text)
				}
			}
		})
	}
}

func TestFormatCodeReviewStarted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		info         CodeReviewStartedInfo
		wantContains []string
	}{
		{
			name: "first iteration",
			info: CodeReviewStartedInfo{
				Iteration:     1,
				MaxIterations: 3,
			},
			wantContains: []string{
				"*Code review phase started*",
				"Review iteration: 1 of 3",
			},
		},
		{
			name: "middle iteration",
			info: CodeReviewStartedInfo{
				Iteration:     2,
				MaxIterations: 5,
			},
			wantContains: []string{
				"*Code review phase started*",
				"Review iteration: 2 of 5",
			},
		},
		{
			name: "final iteration",
			info: CodeReviewStartedInfo{
				Iteration:     3,
				MaxIterations: 3,
			},
			wantContains: []string{
				"*Code review phase started*",
				"Review iteration: 3 of 3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatCodeReviewStarted(tt.info)

			if msg == nil {
				t.Fatal("FormatCodeReviewStarted returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}
		})
	}
}

func TestFormatCodeReviewComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		info         CodeReviewCompleteInfo
		wantContains []string
	}{
		{
			name: "issues found and fixed",
			info: CodeReviewCompleteInfo{
				Iterations:  3,
				IssuesFound: 5,
				IssuesFixed: 4,
				Duration:    10 * time.Minute,
			},
			wantContains: []string{
				"*Code review complete*",
				"Iterations: 3",
				"Issues found: 5",
				"Issues fixed: 4",
				"Duration: 10m 0s",
			},
		},
		{
			name: "no issues found",
			info: CodeReviewCompleteInfo{
				Iterations:  1,
				IssuesFound: 0,
				IssuesFixed: 0,
				Duration:    2 * time.Minute,
			},
			wantContains: []string{
				"*Code review complete*",
				"Iterations: 1",
				"Issues found: 0",
				"Issues fixed: 0",
				"Duration: 2m 0s",
			},
		},
		{
			name: "long review duration",
			info: CodeReviewCompleteInfo{
				Iterations:  5,
				IssuesFound: 10,
				IssuesFixed: 10,
				Duration:    1*time.Hour + 15*time.Minute,
			},
			wantContains: []string{
				"*Code review complete*",
				"Duration: 1h 15m",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatCodeReviewComplete(tt.info)

			if msg == nil {
				t.Fatal("FormatCodeReviewComplete returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}
		})
	}
}

func TestFormatCleanupStarted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		info         CleanupStartedInfo
		wantContains []string
	}{
		{
			name: "single pattern",
			info: CleanupStartedInfo{
				PatternCount: 1,
			},
			wantContains: []string{
				"*Cleanup phase started*",
				"Scanning 1 patterns for artifacts to remove",
			},
		},
		{
			name: "multiple patterns",
			info: CleanupStartedInfo{
				PatternCount: 5,
			},
			wantContains: []string{
				"*Cleanup phase started*",
				"Scanning 5 patterns for artifacts to remove",
			},
		},
		{
			name: "zero patterns",
			info: CleanupStartedInfo{
				PatternCount: 0,
			},
			wantContains: []string{
				"*Cleanup phase started*",
				"Scanning 0 patterns for artifacts to remove",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatCleanupStarted(tt.info)

			if msg == nil {
				t.Fatal("FormatCleanupStarted returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}
		})
	}
}

func TestFormatCleanupComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		info           CleanupCompleteInfo
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "files removed",
			info: CleanupCompleteInfo{
				FilesRemoved: 10,
				Duration:     30 * time.Second,
			},
			wantContains: []string{
				"*Cleanup complete*",
				"Files removed: 10",
				"Duration: 30s",
			},
			wantNotContain: []string{
				"No artifacts found",
			},
		},
		{
			name: "no files to clean",
			info: CleanupCompleteInfo{
				FilesRemoved: 0,
				Duration:     5 * time.Second,
			},
			wantContains: []string{
				"*Cleanup complete*",
				"No artifacts found to clean up",
				"Duration: 5s",
			},
			wantNotContain: []string{
				"Files removed:",
			},
		},
		{
			name: "long cleanup duration",
			info: CleanupCompleteInfo{
				FilesRemoved: 100,
				Duration:     5 * time.Minute,
			},
			wantContains: []string{
				"*Cleanup complete*",
				"Files removed: 100",
				"Duration: 5m 0s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatCleanupComplete(tt.info)

			if msg == nil {
				t.Fatal("FormatCleanupComplete returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(msg.Text, notWant) {
					t.Errorf("message should not contain %q, got:\n%s", notWant, msg.Text)
				}
			}
		})
	}
}

func TestFormatPRCreated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		info         PRCreatedInfo
		wantContains []string
	}{
		{
			name: "standard PR",
			info: PRCreatedInfo{
				PRURL: "https://github.com/user/repo/pull/123",
				Title: "Add new feature",
			},
			wantContains: []string{
				"*Pull request created*",
				"*Title:* Add new feature",
				"<https://github.com/user/repo/pull/123|View PR>",
			},
		},
		{
			name: "long title",
			info: PRCreatedInfo{
				PRURL: "https://github.com/org/project/pull/456",
				Title: "Implement comprehensive user authentication system with OAuth2 support",
			},
			wantContains: []string{
				"*Pull request created*",
				"*Title:* Implement comprehensive user authentication system with OAuth2 support",
				"<https://github.com/org/project/pull/456|View PR>",
			},
		},
		{
			name: "empty title",
			info: PRCreatedInfo{
				PRURL: "https://github.com/user/repo/pull/1",
				Title: "",
			},
			wantContains: []string{
				"*Pull request created*",
				"*Title:* ",
				"<https://github.com/user/repo/pull/1|View PR>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := FormatPRCreated(tt.info)

			if msg == nil {
				t.Fatal("FormatPRCreated returned nil")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg.Text)
				}
			}
		})
	}
}
