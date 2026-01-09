package todo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected *Counts
	}{
		{
			name:    "empty file",
			content: "",
			expected: &Counts{
				Pending:    0,
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "all pending items",
			content: "- [ ] Task 1\n- [ ] Task 2\n- [ ] Task 3\n",
			expected: &Counts{
				Pending:    3,
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "all completed items lowercase x",
			content: "- [x] Task 1\n- [x] Task 2\n",
			expected: &Counts{
				Pending:    0,
				InProgress: 0,
				Completed:  2,
			},
		},
		{
			name:    "all completed items uppercase X",
			content: "- [X] Task 1\n- [X] Task 2\n",
			expected: &Counts{
				Pending:    0,
				InProgress: 0,
				Completed:  2,
			},
		},
		{
			name:    "in progress with dash",
			content: "- [-] Task 1\n- [-] Task 2\n",
			expected: &Counts{
				Pending:    0,
				InProgress: 2,
				Completed:  0,
			},
		},
		{
			name:    "in progress with tilde",
			content: "- [~] Task 1\n- [~] Task 2\n",
			expected: &Counts{
				Pending:    0,
				InProgress: 2,
				Completed:  0,
			},
		},
		{
			name:    "mixed checkbox styles",
			content: "- [ ] Pending\n- [x] Completed lowercase\n- [X] Completed uppercase\n- [-] In progress dash\n- [~] In progress tilde\n",
			expected: &Counts{
				Pending:    1,
				InProgress: 2,
				Completed:  2,
			},
		},
		{
			name:    "file with no checkboxes",
			content: "# Just a heading\n\nSome regular text\n\n- Regular list item\n",
			expected: &Counts{
				Pending:    0,
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "nested lists do not count (indented lines ignored)",
			content: "- [ ] Parent task\n  - [ ] Nested task 1\n  - [x] Nested task 2\n",
			expected: &Counts{
				Pending:    1, // Only parent matches (regex requires ^- at line start)
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "checkboxes with extra spacing",
			content: "-  [ ] Extra space after dash\n- [  ] Extra space in brackets\n",
			expected: &Counts{
				Pending:    2, // Both match (regex allows flexible whitespace)
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "malformed checkboxes",
			content: "[ ] Missing dash\n-[] Missing space after dash\n- [x Missing bracket\n- x] Missing bracket\n",
			expected: &Counts{
				Pending:    1, // -[] matches (regex allows 0 spaces)
				InProgress: 0,
				Completed:  0,
			},
		},
		{
			name:    "checkboxes with content",
			content: "- [ ] First task with description\n- [x] Second task done!\n- [-] Third task in progress...\n",
			expected: &Counts{
				Pending:    1,
				InProgress: 1,
				Completed:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Parse the file
			counts, err := ParseFile(tmpFile)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			// Compare results
			if counts.Pending != tt.expected.Pending {
				t.Errorf("Pending = %d, want %d", counts.Pending, tt.expected.Pending)
			}
			if counts.InProgress != tt.expected.InProgress {
				t.Errorf("InProgress = %d, want %d", counts.InProgress, tt.expected.InProgress)
			}
			if counts.Completed != tt.expected.Completed {
				t.Errorf("Completed = %d, want %d", counts.Completed, tt.expected.Completed)
			}
		})
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	t.Parallel()

	// Non-existent file should return zero counts, not error
	counts, err := ParseFile("/nonexistent/path/TODO.md")
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil for missing file", err)
	}

	if counts.Pending != 0 || counts.InProgress != 0 || counts.Completed != 0 {
		t.Errorf("Expected zero counts for missing file, got Pending=%d, InProgress=%d, Completed=%d",
			counts.Pending, counts.InProgress, counts.Completed)
	}
}

func TestParseItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected []Item
	}{
		{
			name:     "empty file",
			content:  "",
			expected: nil,
		},
		{
			name:    "pending items with text",
			content: "- [ ] First task\n- [ ] Second task\n",
			expected: []Item{
				{Text: "First task", Status: StatusPending},
				{Text: "Second task", Status: StatusPending},
			},
		},
		{
			name:    "completed items with text",
			content: "- [x] Done task 1\n- [X] Done task 2\n",
			expected: []Item{
				{Text: "Done task 1", Status: StatusCompleted},
				{Text: "Done task 2", Status: StatusCompleted},
			},
		},
		{
			name:    "in progress items with text",
			content: "- [-] Working on this\n- [~] Also working\n",
			expected: []Item{
				{Text: "Working on this", Status: StatusInProgress},
				{Text: "Also working", Status: StatusInProgress},
			},
		},
		{
			name:    "mixed statuses",
			content: "- [ ] Pending\n- [x] Completed\n- [-] In Progress\n",
			expected: []Item{
				{Text: "Pending", Status: StatusPending},
				{Text: "Completed", Status: StatusCompleted},
				{Text: "In Progress", Status: StatusInProgress},
			},
		},
		{
			name:    "task with extra whitespace",
			content: "- [ ]   Extra whitespace   \n",
			expected: []Item{
				{Text: "Extra whitespace", Status: StatusPending},
			},
		},
		{
			name:     "no checkboxes returns nil",
			content:  "# Header\n\nJust some text\n",
			expected: nil,
		},
		{
			name:    "nested items not parsed (indented lines ignored)",
			content: "- [ ] Parent\n  - [x] Child 1\n  - [-] Child 2\n",
			expected: []Item{
				{Text: "Parent", Status: StatusPending},
				// Indented lines don't match (regex requires ^- at line start)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Parse the file
			items, err := ParseItems(tmpFile)
			if err != nil {
				t.Fatalf("ParseItems() error = %v", err)
			}

			// Compare lengths
			if len(items) != len(tt.expected) {
				t.Fatalf("Got %d items, want %d", len(items), len(tt.expected))
			}

			// Compare each item
			for i, item := range items {
				if item.Text != tt.expected[i].Text {
					t.Errorf("Item[%d].Text = %q, want %q", i, item.Text, tt.expected[i].Text)
				}
				if item.Status != tt.expected[i].Status {
					t.Errorf("Item[%d].Status = %v, want %v", i, item.Status, tt.expected[i].Status)
				}
			}
		})
	}
}

func TestParseItems_FileNotFound(t *testing.T) {
	t.Parallel()

	// Non-existent file should return nil slice, not error
	items, err := ParseItems("/nonexistent/path/TODO.md")
	if err != nil {
		t.Fatalf("ParseItems() error = %v, want nil for missing file", err)
	}

	if items != nil {
		t.Errorf("Expected nil for missing file, got %v", items)
	}
}

func TestItem_StatusMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		item         Item
		isPending    bool
		isInProgress bool
		isCompleted  bool
	}{
		{
			name:         "pending item",
			item:         Item{Text: "Task", Status: StatusPending},
			isPending:    true,
			isInProgress: false,
			isCompleted:  false,
		},
		{
			name:         "in progress item",
			item:         Item{Text: "Task", Status: StatusInProgress},
			isPending:    false,
			isInProgress: true,
			isCompleted:  false,
		},
		{
			name:         "completed item",
			item:         Item{Text: "Task", Status: StatusCompleted},
			isPending:    false,
			isInProgress: false,
			isCompleted:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.item.IsPending(); got != tt.isPending {
				t.Errorf("IsPending() = %v, want %v", got, tt.isPending)
			}
			if got := tt.item.IsInProgress(); got != tt.isInProgress {
				t.Errorf("IsInProgress() = %v, want %v", got, tt.isInProgress)
			}
			if got := tt.item.IsCompleted(); got != tt.isCompleted {
				t.Errorf("IsCompleted() = %v, want %v", got, tt.isCompleted)
			}
		})
	}
}

func TestCounts_Total(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		counts Counts
		want   int
	}{
		{
			name:   "zero counts",
			counts: Counts{Pending: 0, InProgress: 0, Completed: 0},
			want:   0,
		},
		{
			name:   "only pending",
			counts: Counts{Pending: 5, InProgress: 0, Completed: 0},
			want:   5,
		},
		{
			name:   "only completed",
			counts: Counts{Pending: 0, InProgress: 0, Completed: 3},
			want:   3,
		},
		{
			name:   "pending and completed",
			counts: Counts{Pending: 5, InProgress: 0, Completed: 3},
			want:   8,
		},
		{
			name:   "in progress not counted in total",
			counts: Counts{Pending: 2, InProgress: 3, Completed: 1},
			want:   3, // Total only counts Pending + Completed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.counts.Total(); got != tt.want {
				t.Errorf("Total() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCounts_CompletionRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		counts Counts
		want   float64
	}{
		{
			name:   "zero counts returns 0",
			counts: Counts{Pending: 0, InProgress: 0, Completed: 0},
			want:   0,
		},
		{
			name:   "all pending returns 0",
			counts: Counts{Pending: 5, InProgress: 0, Completed: 0},
			want:   0,
		},
		{
			name:   "all completed returns 100",
			counts: Counts{Pending: 0, InProgress: 0, Completed: 5},
			want:   100,
		},
		{
			name:   "50% completion",
			counts: Counts{Pending: 5, InProgress: 0, Completed: 5},
			want:   50,
		},
		{
			name:   "25% completion",
			counts: Counts{Pending: 3, InProgress: 0, Completed: 1},
			want:   25,
		},
		{
			name:   "in progress not affecting rate",
			counts: Counts{Pending: 4, InProgress: 10, Completed: 4},
			want:   50, // 4/(4+4) = 50%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.counts.CompletionRate()
			if got != tt.want {
				t.Errorf("CompletionRate() = %f, want %f", got, tt.want)
			}
		})
	}
}
