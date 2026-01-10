package github

import (
	"testing"
)

func TestParseIssueRef(t *testing.T) {
	tests := []struct {
		name       string
		ref        string
		wantOwner  string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{
			name:       "issue number only",
			ref:        "42",
			wantOwner:  "",
			wantRepo:   "",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:       "issue number with hash",
			ref:        "#42",
			wantOwner:  "",
			wantRepo:   "",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:       "short reference",
			ref:        "owner/repo#42",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:       "short reference with dashes",
			ref:        "my-org/my-repo#123",
			wantOwner:  "my-org",
			wantRepo:   "my-repo",
			wantNumber: 123,
			wantErr:    false,
		},
		{
			name:       "full URL https",
			ref:        "https://github.com/owner/repo/issues/42",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:       "full URL http",
			ref:        "http://github.com/owner/repo/issues/42",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:       "full URL with dashes",
			ref:        "https://github.com/my-org/my-repo/issues/999",
			wantOwner:  "my-org",
			wantRepo:   "my-repo",
			wantNumber: 999,
			wantErr:    false,
		},
		{
			name:       "whitespace trimmed",
			ref:        "  42  ",
			wantOwner:  "",
			wantRepo:   "",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:    "empty string",
			ref:     "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			ref:     "not-an-issue",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			ref:     "https://github.com/owner/repo/pull/42",
			wantErr: true,
		},
		{
			name:    "incomplete short reference",
			ref:     "owner/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := ParseIssueRef(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIssueRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if owner != tt.wantOwner {
				t.Errorf("ParseIssueRef() owner = %v, want %v", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("ParseIssueRef() repo = %v, want %v", repo, tt.wantRepo)
			}
			if number != tt.wantNumber {
				t.Errorf("ParseIssueRef() number = %v, want %v", number, tt.wantNumber)
			}
		})
	}
}

func TestIssue_Fields(t *testing.T) {
	issue := &Issue{
		Number:    42,
		Title:     "Test Issue",
		Body:      "This is the body",
		Labels:    []string{"bug", "help wanted"},
		URL:       "https://github.com/owner/repo/issues/42",
		Assignees: []string{"user1", "user2"},
		Milestone: "v1.0",
	}

	if issue.Number != 42 {
		t.Errorf("Issue.Number = %v, want 42", issue.Number)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Issue.Title = %v, want 'Test Issue'", issue.Title)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("Issue.Labels length = %v, want 2", len(issue.Labels))
	}
	if len(issue.Assignees) != 2 {
		t.Errorf("Issue.Assignees length = %v, want 2", len(issue.Assignees))
	}
}
