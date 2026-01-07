package config

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const Version = "1.0.0"

// Config holds all configuration for ralph
type Config struct {
	// Core options (matching bash version)
	PromptFile    string
	MaxIterations int
	MaxTime       int
	AgentDir      string
	Cooldown      int
	Verbose       bool
	DryRun        bool

	// OTEL options
	OTELEnabled   bool
	OTELEndpoint  string
	MetricsPrefix string
	ProjectName   string

	// Session info
	SessionID string
}

// DefaultConfig returns a Config with default values matching the bash script
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	return &Config{
		PromptFile:    "./prompt.md",
		MaxIterations: 0,
		MaxTime:       0,
		AgentDir:      "./.agent",
		Cooldown:      1,
		Verbose:       true,
		DryRun:        false,

		OTELEnabled:   false,
		OTELEndpoint:  "localhost:4317",
		MetricsPrefix: "ralph",
		ProjectName:   projectName,

		SessionID: uuid.New().String(),
	}
}

// ScratchpadInstructions returns the instructions appended to prompts
func (c *Config) ScratchpadInstructions() string {
	return "\n\nUse the " + c.AgentDir + " directory as a scratchpad for your work. Keep track of your current status in " + c.AgentDir + "/TODO.md using checkboxes (- [ ] for pending, - [x] for done). Check off items when completed. Only work on a single item at a time and end your session when complete. Make a commit and push your changes after every single file edit."
}
