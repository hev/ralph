package claude

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during a function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestStripSystemReminders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no reminders",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "single reminder",
			input:    "Hello <system-reminder>secret</system-reminder> world",
			expected: "Hello  world",
		},
		{
			name:     "multiple reminders",
			input:    "<system-reminder>first</system-reminder> text <system-reminder>second</system-reminder>",
			expected: "text",
		},
		{
			name:     "multiline reminder",
			input:    "Before\n<system-reminder>\nline1\nline2\n</system-reminder>\nAfter",
			expected: "Before\n\nAfter",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only reminder",
			input:    "<system-reminder>all content</system-reminder>",
			expected: "",
		},
		{
			name:     "nested angle brackets inside",
			input:    "<system-reminder><inner>stuff</inner></system-reminder>",
			expected: "",
		},
		{
			name:     "reminder with whitespace",
			input:    "  <system-reminder>content</system-reminder>  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripSystemReminders(tt.input)
			if result != tt.expected {
				t.Errorf("stripSystemReminders(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "exact length",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "truncation needed",
			input:    "this is a long string",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "zero maxLen",
			input:    "test",
			maxLen:   0,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestIsErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "error type",
			input:    `{"type":"error","message":"something failed"}`,
			expected: true,
		},
		{
			name:     "error field",
			input:    `{"result":"failed","error":"bad request"}`,
			expected: true,
		},
		{
			name:     "no error",
			input:    `{"type":"user","message":"hello"}`,
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "error in text content",
			input:    `{"type":"assistant","message":"The error is fixed"}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("IsErrorMessage(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatReadResult(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxLines int
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			maxLines: 10,
			expected: "",
		},
		{
			name:     "within limit",
			content:  "line1\nline2\nline3",
			maxLines: 5,
			expected: "line1\nline2\nline3",
		},
		{
			name:     "exactly at limit",
			content:  "line1\nline2\nline3",
			maxLines: 3,
			expected: "line1\nline2\nline3",
		},
		{
			name:     "exceeds limit",
			content:  "line1\nline2\nline3\nline4\nline5",
			maxLines: 3,
			expected: "[Read: 5 lines]\nline1\nline2\nline3\n... (2 more lines)",
		},
		{
			name:     "single line",
			content:  "single line",
			maxLines: 1,
			expected: "single line",
		},
		{
			name:     "single line exceeds limit",
			content:  "line1\nline2",
			maxLines: 1,
			expected: "[Read: 2 lines]\nline1\n... (1 more lines)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatReadResult(tt.content, tt.maxLines)
			if result != tt.expected {
				t.Errorf("formatReadResult(%q, %d) = %q, want %q", tt.content, tt.maxLines, result, tt.expected)
			}
		})
	}
}

func TestIsTodoChecklist(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
		{
			name:     "no checkboxes",
			content:  "This is just text\nwith no checkboxes",
			expected: false,
		},
		{
			name:     "single checkbox",
			content:  "- [ ] single item",
			expected: false,
		},
		{
			name:     "two pending checkboxes",
			content:  "- [ ] item 1\n- [ ] item 2",
			expected: true,
		},
		{
			name:     "two completed checkboxes lowercase",
			content:  "- [x] item 1\n- [x] item 2",
			expected: true,
		},
		{
			name:     "two completed checkboxes uppercase",
			content:  "- [X] item 1\n- [X] item 2",
			expected: true,
		},
		{
			name:     "mixed checkboxes",
			content:  "- [ ] pending\n- [x] completed",
			expected: true,
		},
		{
			name:     "checkboxes with header",
			content:  "# TODO\n\n- [ ] item 1\n- [x] item 2",
			expected: true,
		},
		{
			name:     "indented checkboxes",
			content:  "  - [ ] item 1\n  - [ ] item 2",
			expected: true,
		},
		{
			name:     "many checkboxes",
			content:  "- [ ] a\n- [ ] b\n- [x] c\n- [X] d\n- [ ] e",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTodoChecklist(tt.content)
			if result != tt.expected {
				t.Errorf("isTodoChecklist(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestFormatInput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "string input",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "map input",
			input:    map[string]interface{}{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "nested map",
			input:    map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			expected: `{"outer":{"inner":"value"}}`,
		},
		{
			name:     "slice input",
			input:    []string{"a", "b", "c"},
			expected: `["a","b","c"]`,
		},
		{
			name:     "number input",
			input:    42,
			expected: "42",
		},
		{
			name:     "boolean input",
			input:    true,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatInput(tt.input)
			if result != tt.expected {
				t.Errorf("formatInput(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatTodoWrite(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		success  bool
		contains []string
	}{
		{
			name:    "nil input",
			input:   nil,
			success: false,
		},
		{
			name:    "non-map input",
			input:   "string",
			success: false,
		},
		{
			name:    "map without todos",
			input:   map[string]interface{}{"other": "value"},
			success: false,
		},
		{
			name:    "todos not a slice",
			input:   map[string]interface{}{"todos": "not a slice"},
			success: false,
		},
		{
			name: "valid pending todo",
			input: map[string]interface{}{
				"todos": []interface{}{
					map[string]interface{}{"content": "Task 1", "status": "pending"},
				},
			},
			success:  true,
			contains: []string{"Task 1"},
		},
		{
			name: "valid completed todo",
			input: map[string]interface{}{
				"todos": []interface{}{
					map[string]interface{}{"content": "Done task", "status": "completed"},
				},
			},
			success:  true,
			contains: []string{"Done task"},
		},
		{
			name: "valid in_progress todo",
			input: map[string]interface{}{
				"todos": []interface{}{
					map[string]interface{}{"content": "Working on this", "status": "in_progress"},
				},
			},
			success:  true,
			contains: []string{"Working on this"},
		},
		{
			name: "multiple todos",
			input: map[string]interface{}{
				"todos": []interface{}{
					map[string]interface{}{"content": "Task 1", "status": "pending"},
					map[string]interface{}{"content": "Task 2", "status": "completed"},
					map[string]interface{}{"content": "Task 3", "status": "in_progress"},
				},
			},
			success:  true,
			contains: []string{"Task 1", "Task 2", "Task 3"},
		},
		{
			name: "empty todos array",
			input: map[string]interface{}{
				"todos": []interface{}{},
			},
			success: true,
		},
		{
			name: "todo item not a map",
			input: map[string]interface{}{
				"todos": []interface{}{"not a map"},
			},
			success: true, // Still returns true, just skips invalid items
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				result := formatTodoWrite(tt.input)
				if result != tt.success {
					t.Errorf("formatTodoWrite() = %v, want %v", result, tt.success)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("formatTodoWrite() output should contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestFormatEdit(t *testing.T) {
	tests := []struct {
		name         string
		input        interface{}
		expectedPath string
		success      bool
		contains     []string
	}{
		{
			name:    "nil input",
			input:   nil,
			success: false,
		},
		{
			name:    "non-map input",
			input:   "string",
			success: false,
		},
		{
			name:    "missing file_path",
			input:   map[string]interface{}{"old_string": "old", "new_string": "new"},
			success: false,
		},
		{
			name:    "empty file_path",
			input:   map[string]interface{}{"file_path": "", "old_string": "old", "new_string": "new"},
			success: false,
		},
		{
			name: "missing both strings",
			input: map[string]interface{}{
				"file_path":  "/path/to/file.go",
				"old_string": "",
				"new_string": "",
			},
			success: false,
		},
		{
			name: "valid edit with old and new",
			input: map[string]interface{}{
				"file_path":  "/path/to/file.go",
				"old_string": "old content",
				"new_string": "new content",
			},
			expectedPath: "/path/to/file.go",
			success:      true,
			contains:     []string{"old content", "new content"},
		},
		{
			name: "multiline edit",
			input: map[string]interface{}{
				"file_path":  "/path/to/file.go",
				"old_string": "line1\nline2",
				"new_string": "newline1\nnewline2\nnewline3",
			},
			expectedPath: "/path/to/file.go",
			success:      true,
			contains:     []string{"line1", "line2", "newline1", "newline2", "newline3"},
		},
		{
			name: "only old_string",
			input: map[string]interface{}{
				"file_path":  "/path/to/file.go",
				"old_string": "to remove",
				"new_string": "",
			},
			expectedPath: "/path/to/file.go",
			success:      true,
			contains:     []string{"to remove"},
		},
		{
			name: "only new_string",
			input: map[string]interface{}{
				"file_path":  "/path/to/file.go",
				"old_string": "",
				"new_string": "to add",
			},
			expectedPath: "/path/to/file.go",
			success:      true,
			contains:     []string{"to add"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				path, success := formatEdit(tt.input)
				if success != tt.success {
					t.Errorf("formatEdit() success = %v, want %v", success, tt.success)
				}
				if success && path != tt.expectedPath {
					t.Errorf("formatEdit() path = %q, want %q", path, tt.expectedPath)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("formatEdit() output should contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestFormatEditTruncation(t *testing.T) {
	// Create input with more than 20 lines
	var oldLines, newLines []string
	for i := 0; i < 25; i++ {
		oldLines = append(oldLines, "old line")
		newLines = append(newLines, "new line")
	}

	input := map[string]interface{}{
		"file_path":  "/path/to/file.go",
		"old_string": strings.Join(oldLines, "\n"),
		"new_string": strings.Join(newLines, "\n"),
	}

	output := captureOutput(func() {
		path, success := formatEdit(input)
		if !success {
			t.Error("formatEdit() should succeed")
		}
		if path != "/path/to/file.go" {
			t.Errorf("formatEdit() path = %q, want %q", path, "/path/to/file.go")
		}
	})

	// Count old line occurrences - should be exactly 20 (truncated from 25)
	oldCount := strings.Count(output, "old line")
	newCount := strings.Count(output, "new line")
	if oldCount != 20 {
		t.Errorf("formatEdit() should truncate to 20 old lines, got %d", oldCount)
	}
	if newCount != 20 {
		t.Errorf("formatEdit() should truncate to 20 new lines, got %d", newCount)
	}
}

func TestFormatChecklist(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		contains []string
	}{
		{
			name:     "pending items",
			content:  "- [ ] task 1\n- [ ] task 2",
			contains: []string{"task 1", "task 2"},
		},
		{
			name:     "completed items lowercase",
			content:  "- [x] done 1\n- [x] done 2",
			contains: []string{"done 1", "done 2"},
		},
		{
			name:     "completed items uppercase",
			content:  "- [X] DONE 1\n- [X] DONE 2",
			contains: []string{"DONE 1", "DONE 2"},
		},
		{
			name:     "mixed items",
			content:  "# Header\n- [ ] pending\n- [x] done",
			contains: []string{"Header", "pending", "done"},
		},
		{
			name:     "empty lines",
			content:  "- [ ] item\n\n- [x] another",
			contains: []string{"item", "another"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				formatChecklist(tt.content)
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("formatChecklist() output should contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestParseAndPrint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		success  bool
		contains []string
	}{
		{
			name:    "empty line",
			input:   "",
			success: false,
		},
		{
			name:    "invalid JSON",
			input:   "not valid json",
			success: false,
		},
		{
			name:    "user message",
			input:   `{"type":"user","message":{"content":[{"type":"text","text":"Hello"}]}}`,
			success: true,
			contains: []string{"Hello"},
		},
		{
			name:    "user message with system reminder",
			input:   `{"type":"user","message":{"content":[{"type":"text","text":"Hello <system-reminder>secret</system-reminder> world"}]}}`,
			success: true,
			contains: []string{"Hello", "world"},
		},
		{
			name:    "assistant message with text",
			input:   `{"type":"assistant","message":{"content":[{"type":"text","text":"I can help with that"}]}}`,
			success: true,
			contains: []string{"I can help with that"},
		},
		{
			name:    "assistant message with tool_use",
			input:   `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"path":"/file.go"}}]}}`,
			success: true,
			contains: []string{"/file.go"},
		},
		{
			name:    "assistant message with TodoWrite tool",
			input:   `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"TodoWrite","input":{"todos":[{"content":"Task 1","status":"pending"}]}}]}}`,
			success: true,
			contains: []string{"Task 1"},
		},
		{
			name:    "assistant message with Edit tool",
			input:   `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/test.go","old_string":"old","new_string":"new"}}]}}`,
			success: true,
			contains: []string{"old", "new"}, // file_path is printed with color, captured separately
		},
		{
			name:    "result message",
			input:   `{"type":"result","subtype":"success","result":"Operation completed"}`,
			success: true,
			contains: []string{"success", "Operation completed"},
		},
		{
			name:    "result message with system reminder",
			input:   `{"type":"result","result":"Done <system-reminder>hidden</system-reminder> task"}`,
			success: true,
			contains: []string{"Done", "task"},
		},
		{
			name:    "system message with content",
			input:   `{"type":"system","message":{"content":[{"type":"text","text":"System notification"}]}}`,
			success: true,
			contains: []string{"System notification"},
		},
		{
			name:    "system message with subtype",
			input:   `{"type":"system","subtype":"warning"}`,
			success: true,
			contains: []string{"warning"},
		},
		{
			name:    "unknown type",
			input:   `{"type":"unknown","data":"something"}`,
			success: true,
		},
		{
			name:    "user message with tool_result",
			input:   `{"type":"user","message":{"content":[{"type":"tool_result","content":"File contents here"}]}}`,
			success: true,
			contains: []string{"Tool Result", "File contents here"},
		},
		{
			name:    "user message with tool_result checklist",
			input:   `{"type":"user","message":{"content":[{"type":"tool_result","content":"- [ ] item 1\n- [x] item 2"}]}}`,
			success: true,
			contains: []string{"checklist", "item 1", "item 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				result := ParseAndPrint(tt.input)
				if result != tt.success {
					t.Errorf("ParseAndPrint() = %v, want %v", result, tt.success)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("ParseAndPrint() output should contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestParseAndPrintDoesNotContainSystemReminders(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		forbidden []string
	}{
		{
			name:      "user message strips reminders",
			input:     `{"type":"user","message":{"content":[{"type":"text","text":"Hello <system-reminder>secret data</system-reminder>"}]}}`,
			forbidden: []string{"secret data", "system-reminder"},
		},
		{
			name:      "tool result strips reminders",
			input:     `{"type":"user","message":{"content":[{"type":"tool_result","content":"Result <system-reminder>internal info</system-reminder>"}]}}`,
			forbidden: []string{"internal info", "system-reminder"},
		},
		{
			name:      "result message strips reminders",
			input:     `{"type":"result","result":"Done <system-reminder>classified</system-reminder>"}`,
			forbidden: []string{"classified", "system-reminder"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				ParseAndPrint(tt.input)
			})

			for _, forbidden := range tt.forbidden {
				if strings.Contains(output, forbidden) {
					t.Errorf("ParseAndPrint() output should NOT contain %q, got %q", forbidden, output)
				}
			}
		})
	}
}

func TestParseAndPrintResultTruncation(t *testing.T) {
	// Create a result message with more than 1000 characters
	longResult := strings.Repeat("x", 1500)
	input := `{"type":"result","result":"` + longResult + `"}`

	output := captureOutput(func() {
		result := ParseAndPrint(input)
		if !result {
			t.Error("ParseAndPrint() should succeed")
		}
	})

	// The output should be truncated and have ellipsis
	if len(output) > 1200 { // Allow some margin for headers and formatting
		t.Errorf("ParseAndPrint() output should be truncated, got length %d", len(output))
	}
	if !strings.Contains(output, "...") {
		t.Error("ParseAndPrint() truncated output should contain ellipsis")
	}
}

func TestParseAndPrintToolUseTruncation(t *testing.T) {
	// Create an assistant message with a tool_use with very long input
	longInput := strings.Repeat("y", 600)
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Unknown","input":"` + longInput + `"}]}}`

	output := captureOutput(func() {
		result := ParseAndPrint(input)
		if !result {
			t.Error("ParseAndPrint() should succeed")
		}
	})

	// The output should be truncated
	if strings.Contains(output, strings.Repeat("y", 600)) {
		t.Error("ParseAndPrint() should truncate long tool input")
	}
	if !strings.Contains(output, "...") {
		t.Error("ParseAndPrint() truncated output should contain ellipsis")
	}
}

func TestUserMessageToolResultOutput(t *testing.T) {
	// Test that tool_result uses Output field when Content is empty
	input := `{"type":"user","message":{"content":[{"type":"tool_result","output":"Output field value"}]}}`

	output := captureOutput(func() {
		result := ParseAndPrint(input)
		if !result {
			t.Error("ParseAndPrint() should succeed")
		}
	})

	if !strings.Contains(output, "Output field value") {
		t.Errorf("ParseAndPrint() should use Output field, got %q", output)
	}
}

func TestUserMessageToolResultDefaultDone(t *testing.T) {
	// Test that tool_result shows "done" when both Content and Output are empty
	input := `{"type":"user","message":{"content":[{"type":"tool_result"}]}}`

	output := captureOutput(func() {
		result := ParseAndPrint(input)
		if !result {
			t.Error("ParseAndPrint() should succeed")
		}
	})

	if !strings.Contains(output, "done") {
		t.Errorf("ParseAndPrint() should show 'done' for empty tool_result, got %q", output)
	}
}
