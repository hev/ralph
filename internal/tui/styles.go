package tui

import "github.com/charmbracelet/lipgloss"

// Colors matching the existing fatih/color scheme used in parser.go and runner.go
var (
	colorBlue    = lipgloss.Color("12")  // FgBlue
	colorGreen   = lipgloss.Color("10")  // FgGreen
	colorYellow  = lipgloss.Color("11")  // FgYellow
	colorRed     = lipgloss.Color("9")   // FgRed
	colorDim     = lipgloss.Color("240") // Faint/dim gray
	colorWhite   = lipgloss.Color("15")  // Bright white
	colorCyan    = lipgloss.Color("14")  // Cyan for accents
	colorMagenta = lipgloss.Color("13")  // Magenta for highlights
)

// Log line styles for different content types
var (
	// LogDefault is the default style for log lines
	LogDefault = lipgloss.NewStyle()

	// LogInfo is for informational messages (blue)
	LogInfo = lipgloss.NewStyle().Foreground(colorBlue)

	// LogSuccess is for success messages (green)
	LogSuccess = lipgloss.NewStyle().Foreground(colorGreen)

	// LogWarning is for warning messages (yellow)
	LogWarning = lipgloss.NewStyle().Foreground(colorYellow)

	// LogError is for error messages (red)
	LogError = lipgloss.NewStyle().Foreground(colorRed)

	// LogDim is for secondary/muted text
	LogDim = lipgloss.NewStyle().Foreground(colorDim)

	// LogHighlight is for emphasized text
	LogHighlight = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
)

// Footer styles for the pinned status area
var (
	// FooterContainer is the style for the entire footer box
	FooterContainer = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(colorDim)

	// FooterPhase is the style for the phase indicator (e.g., "Main Loop")
	FooterPhase = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	// FooterPaused is the style for the [PAUSED] indicator
	FooterPaused = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	// FooterLabel is the style for labels like "Todos:", "Context:"
	FooterLabel = lipgloss.NewStyle().
			Foreground(colorWhite)

	// FooterValue is the style for values like "5/12 (42%)"
	FooterValue = lipgloss.NewStyle().
			Foreground(colorDim)

	// FooterTodo is the style for the current todo text
	FooterTodo = lipgloss.NewStyle().
			Foreground(colorBlue).
			Italic(true)

	// ProgressBarFilled is the style for filled portions of progress bars
	ProgressBarFilled = lipgloss.NewStyle().
				Foreground(colorGreen)

	// ProgressBarEmpty is the style for empty portions of progress bars
	ProgressBarEmpty = lipgloss.NewStyle().
				Foreground(colorDim)

	// ProgressBarWarning is for when usage is high (>75%)
	ProgressBarWarning = lipgloss.NewStyle().
				Foreground(colorYellow)

	// ProgressBarCritical is for when usage is very high (>90%)
	ProgressBarCritical = lipgloss.NewStyle().
				Foreground(colorRed)
)

// Progress bar characters
const (
	ProgressFilled = "█"
	ProgressEmpty  = "░"
)

// LineType represents the type of log line for styling
type LineType int

const (
	LineTypeDefault LineType = iota
	LineTypeInfo
	LineTypeSuccess
	LineTypeWarning
	LineTypeError
	LineTypeDim
	LineTypeHighlight
)

// StyleForLineType returns the appropriate style for a line type
func StyleForLineType(lt LineType) lipgloss.Style {
	switch lt {
	case LineTypeInfo:
		return LogInfo
	case LineTypeSuccess:
		return LogSuccess
	case LineTypeWarning:
		return LogWarning
	case LineTypeError:
		return LogError
	case LineTypeDim:
		return LogDim
	case LineTypeHighlight:
		return LogHighlight
	default:
		return LogDefault
	}
}
