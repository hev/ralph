package claude

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// LineType represents the type of output line for styling
type LineType int

const (
	LineTypeDefault LineType = iota
	LineTypeInfo             // Blue - user messages, headers
	LineTypeSuccess          // Green - assistant messages, completed items
	LineTypeWarning          // Yellow - tools, results
	LineTypeError            // Red - system messages, errors
	LineTypeDim              // Dim - secondary text
)

// ParsedLine represents a single formatted line of output
type ParsedLine struct {
	Content  string   // The text content of the line
	LineType LineType // The styling type for this line
}

// ParsedOutput represents structured output from parsing a JSON message
type ParsedOutput struct {
	Lines []ParsedLine // All output lines from this message
	Valid bool         // Whether the message was valid JSON
}

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

// ParseOutput parses a JSON line and returns structured output data.
// This is the primary parsing function for TUI integration.
func ParseOutput(line string) ParsedOutput {
	if line == "" {
		return ParsedOutput{Valid: false}
	}

	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		// Not valid JSON, return as raw line
		return ParsedOutput{
			Lines: []ParsedLine{{Content: line, LineType: LineTypeDefault}},
			Valid: false,
		}
	}

	var lines []ParsedLine

	switch msg.Type {
	case "user":
		lines = parseUserMessage(&msg)
	case "assistant":
		lines = parseAssistantMessage(&msg)
	case "result":
		lines = parseResultMessage(&msg)
	case "system":
		lines = parseSystemMessage(&msg)
	default:
		// Other message types as compact JSON
		if line != "" {
			var compact interface{}
			if err := json.Unmarshal([]byte(line), &compact); err == nil {
				out, _ := json.Marshal(compact)
				lines = append(lines, ParsedLine{Content: string(out), LineType: LineTypeDefault})
			} else {
				lines = append(lines, ParsedLine{Content: line, LineType: LineTypeDefault})
			}
		}
	}

	return ParsedOutput{
		Lines: lines,
		Valid: true,
	}
}

// parseUserMessage returns structured lines for a user message
func parseUserMessage(msg *Message) []ParsedLine {
	var lines []ParsedLine
	lines = append(lines, ParsedLine{Content: "━━━ USER ━━━", LineType: LineTypeInfo})

	if msg.Message != nil {
		for _, block := range msg.Message.Content {
			switch block.Type {
			case "text":
				text := stripSystemReminders(block.Text)
				if text != "" {
					lines = append(lines, ParsedLine{Content: text, LineType: LineTypeDefault})
				}
			case "tool_result":
				content := block.Content
				if content == "" {
					content = block.Output
				}
				if content == "" {
					content = "done"
				}
				content = stripSystemReminders(content)

				if isTodoChecklist(content) {
					lines = append(lines, ParsedLine{Content: "Tool Result (checklist):", LineType: LineTypeDefault})
					lines = append(lines, formatChecklistLines(content)...)
				} else {
					content = formatReadResult(content, 10)
					lines = append(lines, ParsedLine{Content: fmt.Sprintf("Tool Result: %s", content), LineType: LineTypeDefault})
				}
			}
		}
	}

	return lines
}

// parseAssistantMessage returns structured lines for an assistant message
func parseAssistantMessage(msg *Message) []ParsedLine {
	var lines []ParsedLine
	lines = append(lines, ParsedLine{Content: "━━━ ASSISTANT ━━━", LineType: LineTypeSuccess})

	if msg.Message != nil {
		for _, block := range msg.Message.Content {
			switch block.Type {
			case "text":
				lines = append(lines, ParsedLine{Content: block.Text, LineType: LineTypeDefault})
			case "tool_use":
				switch block.Name {
				case "TodoWrite":
					lines = append(lines, ParsedLine{Content: "[TOOL: TodoWrite]", LineType: LineTypeWarning})
					todoLines := formatTodoWriteLines(block.Input)
					if len(todoLines) > 0 {
						lines = append(lines, todoLines...)
					} else {
						inputStr := formatInput(block.Input)
						if len(inputStr) > 500 {
							inputStr = inputStr[:500] + "..."
						}
						lines = append(lines, ParsedLine{Content: inputStr, LineType: LineTypeDefault})
					}
				case "Edit":
					if filePath, editLines, ok := formatEditLines(block.Input); ok {
						lines = append(lines, ParsedLine{Content: fmt.Sprintf("[TOOL: Edit] %s", filePath), LineType: LineTypeWarning})
						lines = append(lines, editLines...)
					} else {
						lines = append(lines, ParsedLine{Content: "[TOOL: Edit]", LineType: LineTypeWarning})
						inputStr := formatInput(block.Input)
						if len(inputStr) > 500 {
							inputStr = inputStr[:500] + "..."
						}
						lines = append(lines, ParsedLine{Content: inputStr, LineType: LineTypeDefault})
					}
				default:
					lines = append(lines, ParsedLine{Content: fmt.Sprintf("[TOOL: %s]", block.Name), LineType: LineTypeWarning})
					inputStr := formatInput(block.Input)
					if len(inputStr) > 500 {
						inputStr = inputStr[:500] + "..."
					}
					lines = append(lines, ParsedLine{Content: inputStr, LineType: LineTypeDefault})
				}
			}
		}
	}

	return lines
}

// parseResultMessage returns structured lines for a result message
func parseResultMessage(msg *Message) []ParsedLine {
	var lines []ParsedLine
	lines = append(lines, ParsedLine{Content: "━━━ RESULT ━━━", LineType: LineTypeWarning})

	if msg.Subtype != "" {
		lines = append(lines, ParsedLine{Content: msg.Subtype, LineType: LineTypeDefault})
	}
	if msg.Result != "" {
		result := stripSystemReminders(msg.Result)
		if len(result) > 1000 {
			result = result[:1000] + "..."
		}
		if result != "" {
			lines = append(lines, ParsedLine{Content: result, LineType: LineTypeDefault})
		}
	}

	return lines
}

// parseSystemMessage returns structured lines for a system message
func parseSystemMessage(msg *Message) []ParsedLine {
	var lines []ParsedLine
	lines = append(lines, ParsedLine{Content: "━━━ SYSTEM ━━━", LineType: LineTypeError})

	if msg.Message != nil && len(msg.Message.Content) > 0 {
		for _, block := range msg.Message.Content {
			if block.Text != "" {
				lines = append(lines, ParsedLine{Content: block.Text, LineType: LineTypeDefault})
			}
		}
	} else if msg.Subtype != "" {
		lines = append(lines, ParsedLine{Content: msg.Subtype, LineType: LineTypeDefault})
	}

	return lines
}

// formatTodoWriteLines formats TodoWrite tool input as structured lines
func formatTodoWriteLines(input interface{}) []ParsedLine {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil
	}

	todos, ok := inputMap["todos"].([]interface{})
	if !ok {
		return nil
	}

	var lines []ParsedLine
	for _, item := range todos {
		todoItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		content, _ := todoItem["content"].(string)
		status, _ := todoItem["status"].(string)

		var lineType LineType
		var prefix string
		switch status {
		case "completed":
			prefix = "  ✓ "
			lineType = LineTypeSuccess
		case "in_progress":
			prefix = "  ▶ "
			lineType = LineTypeWarning
		default:
			prefix = "  ○ "
			lineType = LineTypeDim
		}
		lines = append(lines, ParsedLine{Content: prefix + content, LineType: lineType})
	}

	return lines
}

// formatEditLines formats Edit tool input as structured diff lines
func formatEditLines(input interface{}) (string, []ParsedLine, bool) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return "", nil, false
	}

	filePath, _ := inputMap["file_path"].(string)
	oldString, _ := inputMap["old_string"].(string)
	newString, _ := inputMap["new_string"].(string)

	if filePath == "" || (oldString == "" && newString == "") {
		return "", nil, false
	}

	var lines []ParsedLine

	oldLines := strings.Split(oldString, "\n")
	newLines := strings.Split(newString, "\n")

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
		lines = append(lines, ParsedLine{Content: "  - " + line, LineType: LineTypeError})
	}
	if oldTruncated {
		lines = append(lines, ParsedLine{Content: "  ... (truncated)", LineType: LineTypeDim})
	}

	for _, line := range newLines {
		lines = append(lines, ParsedLine{Content: "  + " + line, LineType: LineTypeSuccess})
	}
	if newTruncated {
		lines = append(lines, ParsedLine{Content: "  ... (truncated)", LineType: LineTypeDim})
	}

	return filePath, lines, true
}

// formatChecklistLines formats markdown checklist content as structured lines
func formatChecklistLines(content string) []ParsedLine {
	var lines []ParsedLine
	contentLines := strings.Split(content, "\n")

	for _, line := range contentLines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
			text := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x]"), "- [X]")
			text = strings.TrimSpace(text)
			lines = append(lines, ParsedLine{Content: "  ✓ " + text, LineType: LineTypeSuccess})
		} else if strings.HasPrefix(trimmed, "- [ ]") {
			text := strings.TrimPrefix(trimmed, "- [ ]")
			text = strings.TrimSpace(text)
			lines = append(lines, ParsedLine{Content: "  ○ " + text, LineType: LineTypeDim})
		} else if trimmed != "" {
			lines = append(lines, ParsedLine{Content: "  " + trimmed, LineType: LineTypeDefault})
		}
	}

	return lines
}

// printLine prints a single ParsedLine with appropriate coloring
func printLine(line ParsedLine) {
	switch line.LineType {
	case LineTypeInfo:
		blue.Println(line.Content)
	case LineTypeSuccess:
		green.Println(line.Content)
	case LineTypeWarning:
		yellow.Println(line.Content)
	case LineTypeError:
		red.Println(line.Content)
	case LineTypeDim:
		dim.Println(line.Content)
	default:
		fmt.Println(line.Content)
	}
}

// ParseAndPrint parses a JSON line and prints formatted output
// Returns true if the line was parsed successfully
// This function calls ParseOutput internally for backwards compatibility
func ParseAndPrint(line string) bool {
	if line == "" {
		return false
	}

	output := ParseOutput(line)

	for _, parsedLine := range output.Lines {
		printLine(parsedLine)
	}

	return output.Valid
}

// TUILineType is an alias for passing line types to TUI without circular imports
type TUILineType int

const (
	TUILineTypeDefault TUILineType = iota
	TUILineTypeInfo
	TUILineTypeSuccess
	TUILineTypeWarning
	TUILineTypeError
	TUILineTypeDim
	TUILineTypeHighlight
)

// LineWriter is a callback function for writing lines to a destination
type LineWriter func(content string, lineType TUILineType)

// mapToTUILineType converts claude LineType to TUILineType for TUI integration
func mapToTUILineType(lt LineType) TUILineType {
	switch lt {
	case LineTypeInfo:
		return TUILineTypeInfo
	case LineTypeSuccess:
		return TUILineTypeSuccess
	case LineTypeWarning:
		return TUILineTypeWarning
	case LineTypeError:
		return TUILineTypeError
	case LineTypeDim:
		return TUILineTypeDim
	default:
		return TUILineTypeDefault
	}
}

// ParseAndPrintTo parses a JSON line and writes formatted output via the writer callback.
// This allows routing output to TUI or other destinations.
// Returns true if the line was valid JSON.
func ParseAndPrintTo(line string, writer LineWriter) bool {
	if line == "" {
		return false
	}

	output := ParseOutput(line)

	for _, parsedLine := range output.Lines {
		writer(parsedLine.Content, mapToTUILineType(parsedLine.LineType))
	}

	return output.Valid
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

// isTodoChecklist checks if content looks like a markdown checklist (TODO.md)
// Returns true if the content contains markdown checkbox patterns
func isTodoChecklist(content string) bool {
	if content == "" {
		return false
	}

	// Count lines that match checkbox patterns
	lines := strings.Split(content, "\n")
	checkboxCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
			checkboxCount++
		}
	}

	// Consider it a checklist if at least 2 checkbox items found
	return checkboxCount >= 2
}

