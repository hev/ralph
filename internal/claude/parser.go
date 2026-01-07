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
				fmt.Println(block.Text)
			case "tool_result":
				content := block.Content
				if content == "" {
					content = block.Output
				}
				if content == "" {
					content = "done"
				}
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

func printResultMessage(msg *Message) {
	yellow.Println("━━━ RESULT ━━━")
	if msg.Subtype != "" {
		fmt.Println(msg.Subtype)
	}
	if msg.Result != "" {
		result := msg.Result
		if len(result) > 1000 {
			result = result[:1000] + "..."
		}
		fmt.Println(result)
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
