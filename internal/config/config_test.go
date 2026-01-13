package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	// Core options
	if cfg.PromptFile != "./.agent/IMPLEMENTATION_PLAN.md" {
		t.Errorf("PromptFile = %q, want %q", cfg.PromptFile, "./.agent/IMPLEMENTATION_PLAN.md")
	}
	if cfg.MaxIterations != 0 {
		t.Errorf("MaxIterations = %d, want 0", cfg.MaxIterations)
	}
	if cfg.MaxTime != 0 {
		t.Errorf("MaxTime = %d, want 0", cfg.MaxTime)
	}
	if cfg.AgentDir != "./.agent" {
		t.Errorf("AgentDir = %q, want %q", cfg.AgentDir, "./.agent")
	}
	if cfg.Cooldown != 0 {
		t.Errorf("Cooldown = %d, want 0", cfg.Cooldown)
	}
	if !cfg.Verbose {
		t.Error("Verbose = false, want true")
	}
	if cfg.DryRun {
		t.Error("DryRun = true, want false")
	}

	// OTEL options
	if cfg.OTELEnabled {
		t.Error("OTELEnabled = true, want false")
	}
	if cfg.OTELEndpoint != "localhost:4317" {
		t.Errorf("OTELEndpoint = %q, want %q", cfg.OTELEndpoint, "localhost:4317")
	}
	if cfg.MetricsPrefix != "ralph" {
		t.Errorf("MetricsPrefix = %q, want %q", cfg.MetricsPrefix, "ralph")
	}

	// Slack options
	if cfg.SlackEnabled {
		t.Error("SlackEnabled = true, want false")
	}

	// Code review options
	if cfg.CodeReviewEnabled {
		t.Error("CodeReviewEnabled = true, want false")
	}
	if cfg.CodeReviewMaxIterations != 0 {
		t.Errorf("CodeReviewMaxIterations = %d, want 0 (unlimited)", cfg.CodeReviewMaxIterations)
	}

	// Cleanup options
	if cfg.CleanupEnabled {
		t.Error("CleanupEnabled = true, want false")
	}
	if len(cfg.CleanupPatterns) == 0 {
		t.Error("CleanupPatterns is empty, want default patterns")
	}

	// Worktree options
	if cfg.WorktreeEnabled {
		t.Error("WorktreeEnabled = true, want false")
	}
	if cfg.WorktreeBaseDir != "/tmp/ralph-worktrees" {
		t.Errorf("WorktreeBaseDir = %q, want %q", cfg.WorktreeBaseDir, "/tmp/ralph-worktrees")
	}
	if cfg.WorktreeBranchPrefix != "ralph/" {
		t.Errorf("WorktreeBranchPrefix = %q, want %q", cfg.WorktreeBranchPrefix, "ralph/")
	}
	if !cfg.WorktreeCleanup {
		t.Error("WorktreeCleanup = false, want true")
	}

	// PR options
	if cfg.PREnabled {
		t.Error("PREnabled = true, want false")
	}

	// Session ID should be set
	if cfg.SessionID == "" {
		t.Error("SessionID is empty, want a UUID")
	}
}

func TestLoadFromFile_ValidYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "minimal yaml with prompt",
			yaml: `prompt: ./custom-prompt.md`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.PromptFile != "./custom-prompt.md" {
					t.Errorf("PromptFile = %q, want %q", cfg.PromptFile, "./custom-prompt.md")
				}
			},
		},
		{
			name: "max iterations and time",
			yaml: `
max_iterations: 10
max_time: 3600
`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.MaxIterations != 10 {
					t.Errorf("MaxIterations = %d, want 10", cfg.MaxIterations)
				}
				if cfg.MaxTime != 3600 {
					t.Errorf("MaxTime = %d, want 3600", cfg.MaxTime)
				}
			},
		},
		{
			name: "boolean options",
			yaml: `
verbose: false
dry_run: true
`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Verbose {
					t.Error("Verbose = true, want false")
				}
				if !cfg.DryRun {
					t.Error("DryRun = false, want true")
				}
			},
		},
		{
			name: "model option",
			yaml: `model: opus`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Model != "opus" {
					t.Errorf("Model = %q, want %q", cfg.Model, "opus")
				}
			},
		},
		{
			name: "otel options",
			yaml: `
otel:
  enabled: true
  endpoint: custom:4317
  metrics_prefix: myapp
  project_name: testproject
`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.OTELEnabled {
					t.Error("OTELEnabled = false, want true")
				}
				if cfg.OTELEndpoint != "custom:4317" {
					t.Errorf("OTELEndpoint = %q, want %q", cfg.OTELEndpoint, "custom:4317")
				}
				if cfg.MetricsPrefix != "myapp" {
					t.Errorf("MetricsPrefix = %q, want %q", cfg.MetricsPrefix, "myapp")
				}
				if cfg.ProjectName != "testproject" {
					t.Errorf("ProjectName = %q, want %q", cfg.ProjectName, "testproject")
				}
			},
		},
		{
			name: "slack options",
			yaml: `
slack:
  enabled: true
  webhook_url: https://hooks.slack.com/test
  bot_token: xoxb-token
  channel: C123456
  notify_users: U123,U456
`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.SlackEnabled {
					t.Error("SlackEnabled = false, want true")
				}
				if cfg.SlackWebhookURL != "https://hooks.slack.com/test" {
					t.Errorf("SlackWebhookURL = %q, want webhook URL", cfg.SlackWebhookURL)
				}
				if cfg.SlackBotToken != "xoxb-token" {
					t.Errorf("SlackBotToken = %q, want %q", cfg.SlackBotToken, "xoxb-token")
				}
				if cfg.SlackChannel != "C123456" {
					t.Errorf("SlackChannel = %q, want %q", cfg.SlackChannel, "C123456")
				}
				if cfg.SlackNotifyUsers != "U123,U456" {
					t.Errorf("SlackNotifyUsers = %q, want %q", cfg.SlackNotifyUsers, "U123,U456")
				}
			},
		},
		{
			name: "stop on completion",
			yaml: `stop_on_completion: true`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.StopOnCompletion {
					t.Error("StopOnCompletion = false, want true")
				}
			},
		},
		{
			name: "code review options",
			yaml: `
code_review:
  enabled: true
  max_iterations: 5
  prompt: "Custom review prompt"
  model: opus
`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.CodeReviewEnabled {
					t.Error("CodeReviewEnabled = false, want true")
				}
				if cfg.CodeReviewMaxIterations != 5 {
					t.Errorf("CodeReviewMaxIterations = %d, want 5", cfg.CodeReviewMaxIterations)
				}
				if cfg.CodeReviewPrompt != "Custom review prompt" {
					t.Errorf("CodeReviewPrompt = %q, want custom prompt", cfg.CodeReviewPrompt)
				}
				if cfg.CodeReviewModel != "opus" {
					t.Errorf("CodeReviewModel = %q, want %q", cfg.CodeReviewModel, "opus")
				}
			},
		},
		{
			name: "cleanup options",
			yaml: `
cleanup:
  enabled: true
  patterns:
    - "*.tmp"
    - "*.bak"
  model: haiku
`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.CleanupEnabled {
					t.Error("CleanupEnabled = false, want true")
				}
				if len(cfg.CleanupPatterns) != 2 {
					t.Errorf("CleanupPatterns length = %d, want 2", len(cfg.CleanupPatterns))
				} else {
					if cfg.CleanupPatterns[0] != "*.tmp" {
						t.Errorf("CleanupPatterns[0] = %q, want %q", cfg.CleanupPatterns[0], "*.tmp")
					}
					if cfg.CleanupPatterns[1] != "*.bak" {
						t.Errorf("CleanupPatterns[1] = %q, want %q", cfg.CleanupPatterns[1], "*.bak")
					}
				}
				if cfg.CleanupModel != "haiku" {
					t.Errorf("CleanupModel = %q, want %q", cfg.CleanupModel, "haiku")
				}
			},
		},
		{
			name: "worktree options",
			yaml: `
worktree:
  enabled: true
  base_dir: /custom/worktrees
  branch_prefix: feature/
  cleanup: false
`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.WorktreeEnabled {
					t.Error("WorktreeEnabled = false, want true")
				}
				if cfg.WorktreeBaseDir != "/custom/worktrees" {
					t.Errorf("WorktreeBaseDir = %q, want custom path", cfg.WorktreeBaseDir)
				}
				if cfg.WorktreeBranchPrefix != "feature/" {
					t.Errorf("WorktreeBranchPrefix = %q, want %q", cfg.WorktreeBranchPrefix, "feature/")
				}
				if cfg.WorktreeCleanup {
					t.Error("WorktreeCleanup = true, want false")
				}
			},
		},
		{
			name: "pr options",
			yaml: `
pr:
  enabled: true
  title: "Custom PR Title"
  base: develop
`,
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.PREnabled {
					t.Error("PREnabled = false, want true")
				}
				if cfg.PRTitle != "Custom PR Title" {
					t.Errorf("PRTitle = %q, want custom title", cfg.PRTitle)
				}
				if cfg.PRBase != "develop" {
					t.Errorf("PRBase = %q, want %q", cfg.PRBase, "develop")
				}
			},
		},
		{
			name: "empty yaml file preserves defaults",
			yaml: ``,
			validate: func(t *testing.T, cfg *Config) {
				// All defaults should be preserved
				if cfg.PromptFile != "./.agent/IMPLEMENTATION_PLAN.md" {
					t.Errorf("PromptFile = %q, want default", cfg.PromptFile)
				}
				if cfg.MaxIterations != 0 {
					t.Errorf("MaxIterations = %d, want 0", cfg.MaxIterations)
				}
			},
		},
		{
			name: "partial yaml only overwrites specified fields",
			yaml: `
max_iterations: 5
slack:
  enabled: true
`,
			validate: func(t *testing.T, cfg *Config) {
				// Overwritten
				if cfg.MaxIterations != 5 {
					t.Errorf("MaxIterations = %d, want 5", cfg.MaxIterations)
				}
				if !cfg.SlackEnabled {
					t.Error("SlackEnabled = false, want true")
				}
				// Defaults preserved
				if cfg.PromptFile != "./.agent/IMPLEMENTATION_PLAN.md" {
					t.Errorf("PromptFile = %q, want default", cfg.PromptFile)
				}
				if cfg.OTELEnabled {
					t.Error("OTELEnabled = true, want false (default)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "ralph.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Load config
			cfg := DefaultConfig()
			if err := cfg.LoadFromFile(tmpFile); err != nil {
				t.Fatalf("LoadFromFile() error = %v", err)
			}

			// Validate
			tt.validate(t, cfg)
		})
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ralph.yaml")
	if err := os.WriteFile(tmpFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg := DefaultConfig()
	err := cfg.LoadFromFile(tmpFile)
	if err == nil {
		t.Error("LoadFromFile() expected error for invalid YAML, got nil")
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	err := cfg.LoadFromFile("/nonexistent/path/ralph.yaml")
	if err == nil {
		t.Error("LoadFromFile() expected error for missing file, got nil")
	}
}

func TestGetSlackNotifyUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single user",
			input:    "U12345",
			expected: []string{"U12345"},
		},
		{
			name:     "multiple users",
			input:    "U123,U456,U789",
			expected: []string{"U123", "U456", "U789"},
		},
		{
			name:     "users with spaces",
			input:    "U123, U456, U789",
			expected: []string{"U123", "U456", "U789"},
		},
		{
			name:     "users with leading/trailing spaces",
			input:    "  U123  ,  U456  ",
			expected: []string{"U123", "U456"},
		},
		{
			name:     "empty entries filtered out",
			input:    "U123,,U456,",
			expected: []string{"U123", "U456"},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: nil,
		},
		{
			name:     "only spaces",
			input:    "  ,  ,  ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{SlackNotifyUsers: tt.input}
			result := cfg.GetSlackNotifyUsers()

			if len(result) != len(tt.expected) {
				t.Fatalf("Got %d users, want %d", len(result), len(tt.expected))
			}

			for i, user := range result {
				if user != tt.expected[i] {
					t.Errorf("User[%d] = %q, want %q", i, user, tt.expected[i])
				}
			}
		})
	}
}

func TestScratchpadInstructions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		agentDir string
	}{
		{
			name:     "default agent dir",
			agentDir: "./.agent",
		},
		{
			name:     "custom agent dir",
			agentDir: "/custom/agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{AgentDir: tt.agentDir}
			instructions := cfg.ScratchpadInstructions()

			// Check that agent dir is in instructions
			if instructions == "" {
				t.Error("ScratchpadInstructions() returned empty string")
			}

			// Instructions should contain the agent dir path twice
			if !contains(instructions, tt.agentDir) {
				t.Errorf("Instructions should contain agent dir %q", tt.agentDir)
			}

			// Instructions should mention TODO.md
			if !contains(instructions, "TODO.md") {
				t.Error("Instructions should mention TODO.md")
			}
		})
	}
}

func TestCodeReviewInstructions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		customPrompt  string
		agentDir      string
		expectCustom  bool
		expectContain string
	}{
		{
			name:          "default instructions",
			customPrompt:  "",
			agentDir:      "./.agent",
			expectCustom:  false,
			expectContain: "Review the code changes",
		},
		{
			name:          "custom prompt",
			customPrompt:  "My custom review prompt",
			agentDir:      "./.agent",
			expectCustom:  true,
			expectContain: "My custom review prompt",
		},
		{
			name:          "default includes agent dir",
			customPrompt:  "",
			agentDir:      "/custom/dir",
			expectCustom:  false,
			expectContain: "/custom/dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				CodeReviewPrompt: tt.customPrompt,
				AgentDir:         tt.agentDir,
			}
			instructions := cfg.CodeReviewInstructions()

			if !contains(instructions, tt.expectContain) {
				t.Errorf("Instructions should contain %q", tt.expectContain)
			}

			if tt.expectCustom && instructions != tt.customPrompt {
				t.Errorf("Expected custom prompt, got %q", instructions)
			}
		})
	}
}

func TestFindConfigFiles(t *testing.T) {
	// Note: This test doesn't use t.Parallel() because it changes directories

	// Save original directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	tests := []struct {
		name       string
		setupLocal bool
	}{
		{
			name:       "no local config",
			setupLocal: false,
		},
		{
			name:       "local config exists",
			setupLocal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			if tt.setupLocal {
				if err := os.WriteFile("ralph.yaml", []byte("prompt: test.md"), 0644); err != nil {
					t.Fatalf("Failed to create local config: %v", err)
				}
			}

			paths := FindConfigFiles()

			// Check if local config is in the paths
			localFound := false
			for _, p := range paths {
				// Check if this is the local ralph.yaml we created
				absPath, _ := filepath.Abs("ralph.yaml")
				if p == absPath || p == "ralph.yaml" {
					localFound = true
				}
			}

			if tt.setupLocal && !localFound {
				t.Errorf("Expected local ralph.yaml to be found in paths: %v", paths)
			}
			if !tt.setupLocal && localFound {
				t.Error("Did not expect local ralph.yaml to be found")
			}
		})
	}
}

func TestFindConfigFile_Deprecated(t *testing.T) {
	// Note: This test doesn't use t.Parallel() because it changes directories

	// Save original directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Test behavior without local config
	// Note: may return global config if it exists, so we just check the logic
	pathWithoutLocal := FindConfigFile()

	// Test local config takes precedence
	if err := os.WriteFile("ralph.yaml", []byte("prompt: test.md"), 0644); err != nil {
		t.Fatalf("Failed to create local config: %v", err)
	}

	pathWithLocal := FindConfigFile()
	if pathWithLocal != "ralph.yaml" {
		t.Errorf("Expected local 'ralph.yaml' to take precedence, got %q", pathWithLocal)
	}

	// Local config should be different from what we had before (unless global was also "ralph.yaml")
	if pathWithoutLocal == "ralph.yaml" {
		// This shouldn't happen since we're in a temp dir
		t.Error("Unexpected: found 'ralph.yaml' before creating it")
	}
}

func TestLoadFromFile_ConfigMerging(t *testing.T) {
	t.Parallel()

	// Test that loading multiple files correctly merges config
	// Local values should override global values

	tmpDir := t.TempDir()

	// Create "global" config
	globalConfig := filepath.Join(tmpDir, "global.yaml")
	globalYAML := `
prompt: global-prompt.md
max_iterations: 100
slack:
  enabled: true
  channel: C-global
`
	if err := os.WriteFile(globalConfig, []byte(globalYAML), 0644); err != nil {
		t.Fatalf("Failed to create global config: %v", err)
	}

	// Create "local" config (overrides some values)
	localConfig := filepath.Join(tmpDir, "local.yaml")
	localYAML := `
max_iterations: 10
slack:
  channel: C-local
`
	if err := os.WriteFile(localConfig, []byte(localYAML), 0644); err != nil {
		t.Fatalf("Failed to create local config: %v", err)
	}

	// Load configs in order (global first, then local)
	cfg := DefaultConfig()
	if err := cfg.LoadFromFile(globalConfig); err != nil {
		t.Fatalf("LoadFromFile(global) error = %v", err)
	}
	if err := cfg.LoadFromFile(localConfig); err != nil {
		t.Fatalf("LoadFromFile(local) error = %v", err)
	}

	// Check values
	// prompt should be from global (not overwritten by local)
	if cfg.PromptFile != "global-prompt.md" {
		t.Errorf("PromptFile = %q, want global value", cfg.PromptFile)
	}

	// max_iterations should be from local (overwritten)
	if cfg.MaxIterations != 10 {
		t.Errorf("MaxIterations = %d, want 10 (local value)", cfg.MaxIterations)
	}

	// slack.enabled should be from global (not overwritten by local)
	if !cfg.SlackEnabled {
		t.Error("SlackEnabled = false, want true (global value)")
	}

	// slack.channel should be from local (overwritten)
	if cfg.SlackChannel != "C-local" {
		t.Errorf("SlackChannel = %q, want local value", cfg.SlackChannel)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if Version == "" {
		t.Error("Version constant is empty")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
