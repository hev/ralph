package claude

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

var (
	blue   = color.New(color.FgBlue)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	dim    = color.New(color.Faint)

	// Regex to match <system-reminder>...</system-reminder> blocks
	systemReminderRegex = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)
)

// Message represents a streaming JSON message from Claude
type Message struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message *MessageContent `json:"message,omitempty"`
	Result  string          `json:"result,omitempty"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Content []ContentBlock `json:"content,omitempty"`
}

// ContentBlock represents a single content block
type ContentBlock struct {
	Type    string      `json:"type"`
	Text    string      `json:"text,omitempty"`
	Name    string      `json:"name,omitempty"`
	Input   interface{} `json:"input,omitempty"`
	Content string      `json:"content,omitempty"`
	Output  string      `json:"output,omitempty"`
}

// ParseAndPrint parses a JSON line and prints formatted output
// Returns true if the line was parsed successfully
func ParseAndPrint(line string) bool {
	if line == "" {
		return false
	}

	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		// Not valid JSON, print raw
		fmt.Println(line)
		return false
	}

	switch msg.Type {
	case "user":
		printUserMessage(&msg)
	case "assistant":
		printAssistantMessage(&msg)
	case "result":
		printResultMessage(&msg)
	case "system":
		printSystemMessage(&msg)
	default:
		// Print other message types as compact JSON
		if line != "" {
			var compact interface{}
			if err := json.Unmarshal([]byte(line), &compact); err == nil {
				out, _ := json.Marshal(compact)
				fmt.Println(string(out))
			} else {
				fmt.Println(line)
			}
		}
	}

	return true
}

func printUserMessage(msg *Message) {
	blue.Println("━━━ USER ━━━")
	if msg.Message != nil {
		for _, block := range msg.Message.Content {
			switch block.Type {
			case "text":
				text := stripSystemReminders(block.Text)
				if text != "" {
					fmt.Println(text)
				}
			case "tool_result":
				content := block.Content
				if content == "" {
					content = block.Output
				}
				if content == "" {
					content = "done"
				}
				// Strip system reminders and truncate large results
				content = stripSystemReminders(content)
				content = formatReadResult(content, 10)
				fmt.Printf("Tool Result: %s\n", content)
			}
		}
	}
}

func printAssistantMessage(msg *Message) {
	green.Println("━━━ ASSISTANT ━━━")
	if msg.Message != nil {
		for _, block := range msg.Message.Content {
			switch block.Type {
			case "text":
				fmt.Println(block.Text)
			case "tool_use":
				// Use special formatters for specific tools
				switch block.Name {
				case "TodoWrite":
					yellow.Println("[TOOL: TodoWrite]")
					if !formatTodoWrite(block.Input) {
						// Fallback to default formatting
						inputStr := formatInput(block.Input)
						if len(inputStr) > 500 {
							inputStr = inputStr[:500] + "..."
						}
						fmt.Println(inputStr)
					}
				case "Edit":
					if filePath, ok := formatEdit(block.Input); ok {
						yellow.Printf("[TOOL: Edit] %s\n", filePath)
					} else {
						yellow.Println("[TOOL: Edit]")
						inputStr := formatInput(block.Input)
						if len(inputStr) > 500 {
							inputStr = inputStr[:500] + "..."
						}
						fmt.Println(inputStr)
					}
				default:
					yellow.Printf("[TOOL: %s]\n", block.Name)
					inputStr := formatInput(block.Input)
					if len(inputStr) > 500 {
						inputStr = inputStr[:500] + "..."
					}
					fmt.Println(inputStr)
				}
			}
		}
	}
}

func printResultMessage(msg *Message) {
	yellow.Println("━━━ RESULT ━━━")
	if msg.Subtype != "" {
		fmt.Println(msg.Subtype)
	}
	if msg.Result != "" {
		result := stripSystemReminders(msg.Result)
		if len(result) > 1000 {
			result = result[:1000] + "..."
		}
		if result != "" {
			fmt.Println(result)
		}
	}
}

func printSystemMessage(msg *Message) {
	red.Println("━━━ SYSTEM ━━━")
	if msg.Message != nil && len(msg.Message.Content) > 0 {
		for _, block := range msg.Message.Content {
			if block.Text != "" {
				fmt.Println(block.Text)
			}
		}
	} else if msg.Subtype != "" {
		fmt.Println(msg.Subtype)
	}
}

func formatInput(input interface{}) string {
	if input == nil {
		return ""
	}

	switch v := input.(type) {
	case string:
		return v
	case map[string]interface{}:
		// Try to format as JSON
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// IsErrorMessage checks if the message indicates an error
func IsErrorMessage(line string) bool {
	return strings.Contains(line, "\"type\":\"error\"") ||
		strings.Contains(line, "\"error\":")
}

// stripSystemReminders removes all <system-reminder>...</system-reminder> blocks from text
func stripSystemReminders(text string) string {
	return strings.TrimSpace(systemReminderRegex.ReplaceAllString(text, ""))
}

// formatTodoWrite formats TodoWrite tool input as a checklist with status icons
// Returns true if formatting was successful, false if fallback to default formatting is needed
func formatTodoWrite(input interface{}) bool {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return false
	}

	todos, ok := inputMap["todos"].([]interface{})
	if !ok {
		return false
	}

	for _, item := range todos {
		todoItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		content, _ := todoItem["content"].(string)
		status, _ := todoItem["status"].(string)

		switch status {
		case "completed":
			green.Print("  \u2713 ") // checkmark
			fmt.Println(content)
		case "in_progress":
			yellow.Print("  \u25b6 ") // play/arrow
			fmt.Println(content)
		default: // pending
			dim.Print("  \u25cb ") // circle
			fmt.Println(content)
		}
	}

	return true
}

// formatReadResult formats/truncates Read tool result content
// Returns formatted string with line count summary if truncated
func formatReadResult(content string, maxLines int) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	if totalLines <= maxLines {
		return content
	}

	// Truncate and show summary
	truncated := strings.Join(lines[:maxLines], "\n")
	return fmt.Sprintf("[Read: %d lines]\n%s\n... (%d more lines)", totalLines, truncated, totalLines-maxLines)
}

// formatEdit formats Edit tool input as a colored diff
// Returns the file path (for header display) and true if formatting was successful
func formatEdit(input interface{}) (string, bool) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return "", false
	}

	filePath, _ := inputMap["file_path"].(string)
	oldString, _ := inputMap["old_string"].(string)
	newString, _ := inputMap["new_string"].(string)

	if filePath == "" || (oldString == "" && newString == "") {
		return "", false
	}

	// Print old string lines with - prefix in red
	oldLines := strings.Split(oldString, "\n")
	newLines := strings.Split(newString, "\n")

	// Truncate if too long
	const maxLines = 20
	oldTruncated := len(oldLines) > maxLines
	newTruncated := len(newLines) > maxLines
	if oldTruncated {
		oldLines = oldLines[:maxLines]
	}
	if newTruncated {
		newLines = newLines[:maxLines]
	}

	for _, line := range oldLines {
		red.Print("  - ")
		fmt.Println(line)
	}
	if oldTruncated {
		dim.Println("  ... (truncated)")
	}

	for _, line := range newLines {
		green.Print("  + ")
		fmt.Println(line)
	}
	if newTruncated {
		dim.Println("  ... (truncated)")
	}

	return filePath, true
}
