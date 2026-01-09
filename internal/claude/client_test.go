package claude

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestNewClient_CommandConstruction(t *testing.T) {
	tests := []struct {
		name         string
		prompt       string
		model        string
		wantArgs     []string
		dontWantArgs []string
	}{
		{
			name:   "basic prompt without model",
			prompt: "test prompt",
			model:  "",
			wantArgs: []string{
				"--dangerously-skip-permissions",
				"--print",
				"--verbose",
				"--output-format", "stream-json",
				"-p", "test prompt",
			},
			dontWantArgs: []string{"--model"},
		},
		{
			name:   "prompt with model specified",
			prompt: "another prompt",
			model:  "opus",
			wantArgs: []string{
				"--dangerously-skip-permissions",
				"--print",
				"--verbose",
				"--output-format", "stream-json",
				"--model", "opus",
				"-p", "another prompt",
			},
		},
		{
			name:   "prompt with sonnet model",
			prompt: "sonnet test",
			model:  "sonnet",
			wantArgs: []string{
				"--model", "sonnet",
				"-p", "sonnet test",
			},
		},
		{
			name:   "empty prompt",
			prompt: "",
			model:  "",
			wantArgs: []string{
				"-p", "",
			},
		},
		{
			name:   "prompt with special characters",
			prompt: "test with 'quotes' and \"double quotes\"",
			model:  "",
			wantArgs: []string{
				"-p", "test with 'quotes' and \"double quotes\"",
			},
		},
		{
			name:   "multiline prompt",
			prompt: "line1\nline2\nline3",
			model:  "",
			wantArgs: []string{
				"-p", "line1\nline2\nline3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.prompt, tt.model)
			if err != nil {
				t.Fatalf("NewClient() error = %v, want nil", err)
			}

			if client == nil {
				t.Fatal("NewClient() returned nil client")
			}

			if client.cmd == nil {
				t.Fatal("NewClient() returned client with nil cmd")
			}

			// Check that expected args are present
			args := client.cmd.Args
			argsStr := strings.Join(args, " ")

			for i := 0; i < len(tt.wantArgs); i++ {
				found := false
				for j := 0; j < len(args); j++ {
					if args[j] == tt.wantArgs[i] {
						// If this is a flag that takes a value, check the value too
						if i+1 < len(tt.wantArgs) && !strings.HasPrefix(tt.wantArgs[i+1], "-") {
							if j+1 < len(args) && args[j+1] == tt.wantArgs[i+1] {
								found = true
								i++ // Skip the value in outer loop
								break
							}
						} else {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("NewClient() args missing %q, got: %s", tt.wantArgs[i], argsStr)
				}
			}

			// Check that unwanted args are not present
			for _, unwanted := range tt.dontWantArgs {
				for _, arg := range args {
					if arg == unwanted {
						t.Errorf("NewClient() args should not contain %q, got: %s", unwanted, argsStr)
					}
				}
			}
		})
	}
}

func TestNewClient_CommandName(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test", "")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// First element should be the command name
	if len(client.cmd.Args) == 0 {
		t.Fatal("NewClient() cmd.Args is empty")
	}

	// The path might be full path or just "claude"
	if !strings.HasSuffix(client.cmd.Args[0], "claude") && client.cmd.Args[0] != "claude" {
		t.Errorf("NewClient() command = %q, want command ending with 'claude'", client.cmd.Args[0])
	}
}

func TestNewClient_StdoutStderrPipes(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test", "")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.stdout == nil {
		t.Error("NewClient() stdout pipe is nil")
	}

	if client.stderr == nil {
		t.Error("NewClient() stderr pipe is nil")
	}
}

func TestClient_ImplementsRunner(t *testing.T) {
	// Compile-time check that Client implements Runner
	var _ Runner = (*Client)(nil)
}

func TestClient_Kill_NilProcess(t *testing.T) {
	// Test Kill when process hasn't started
	client := &Client{
		cmd: exec.Command("echo", "test"),
	}

	// Process is nil before Start(), Kill should not panic
	err := client.Kill()
	if err != nil {
		t.Errorf("Kill() with nil process error = %v, want nil", err)
	}
}

func TestClient_StreamOutput_ReturnsChannel(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test", "")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ch := client.StreamOutput()
	if ch == nil {
		t.Error("StreamOutput() returned nil channel")
	}
}

func TestClient_WithContext_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with a real echo command to verify the flow works
	ctx := context.Background()

	// Create a simple command instead of claude to test the flow
	cmd := exec.CommandContext(ctx, "echo", "test output")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	// Start the process
	if err := client.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stream output
	ch := client.StreamOutput()

	// Collect output
	var output []string
	for line := range ch {
		output = append(output, line)
	}

	// Wait for completion
	exitCode, err := client.Wait()
	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Wait() exitCode = %d, want 0", exitCode)
	}

	// Verify we got output
	if len(output) == 0 {
		t.Error("StreamOutput() produced no output")
	}
}

func TestClient_Wait_ExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name         string
		command      string
		args         []string
		wantExitCode int
		wantErr      bool
	}{
		{
			name:         "exit code 0",
			command:      "true",
			args:         nil,
			wantExitCode: 0,
			wantErr:      false,
		},
		{
			name:         "exit code 1",
			command:      "false",
			args:         nil,
			wantExitCode: 1,
			wantErr:      true,
		},
		{
			name:         "exit code from shell",
			command:      "sh",
			args:         []string{"-c", "exit 42"},
			wantExitCode: 42,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cmd := exec.CommandContext(ctx, tt.command, tt.args...)
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()

			client := &Client{
				cmd:    cmd,
				stdout: stdout,
				stderr: stderr,
			}

			if err := client.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			exitCode, err := client.Wait()

			if (err != nil) != tt.wantErr {
				t.Errorf("Wait() error = %v, wantErr %v", err, tt.wantErr)
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Wait() exitCode = %d, want %d", exitCode, tt.wantExitCode)
			}
		})
	}
}

func TestClient_Kill_RunningProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	// Start a long-running process
	cmd := exec.CommandContext(ctx, "sleep", "10")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	if err := client.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give the process time to start
	time.Sleep(100 * time.Millisecond)

	// Kill the process
	if err := client.Kill(); err != nil {
		t.Errorf("Kill() error = %v", err)
	}

	// Wait should return with a signal error
	_, err := client.Wait()
	if err == nil {
		t.Error("Wait() after Kill() should return error")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running process
	cmd := exec.CommandContext(ctx, "sleep", "10")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	if err := client.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give the process time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait should return with an error due to context cancellation
	_, err := client.Wait()
	if err == nil {
		t.Error("Wait() after context cancellation should return error")
	}
}

func TestClient_StreamOutput_LargeOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	// Generate many lines of output
	cmd := exec.CommandContext(ctx, "sh", "-c", "for i in $(seq 1 100); do echo \"line $i\"; done")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	if err := client.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	ch := client.StreamOutput()

	// Collect all output
	var lineCount int
	for range ch {
		lineCount++
	}

	client.Wait()

	if lineCount != 100 {
		t.Errorf("StreamOutput() received %d lines, want 100", lineCount)
	}
}

func TestClient_StreamOutput_EmptyOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	// Command that produces no output
	cmd := exec.CommandContext(ctx, "true")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	if err := client.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	ch := client.StreamOutput()

	// Collect all output
	var lineCount int
	for range ch {
		lineCount++
	}

	client.Wait()

	if lineCount != 0 {
		t.Errorf("StreamOutput() received %d lines, want 0 for empty output", lineCount)
	}
}

func TestClient_Start_AlreadyStarted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "sleep", "1")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	// First start should succeed
	if err := client.Start(); err != nil {
		t.Fatalf("Start() first call error = %v", err)
	}

	// Second start should fail
	err := client.Start()
	if err == nil {
		t.Error("Start() second call should return error")
	}

	// Clean up
	client.Kill()
	client.Wait()
}

func TestClient_Wait_NotStarted(t *testing.T) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "echo", "test")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	client := &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}

	// Wait without Start should return error
	_, err := client.Wait()
	if err == nil {
		t.Error("Wait() without Start() should return error")
	}
}

func TestNewClient_ArgumentOrder(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test prompt", "opus")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	args := client.cmd.Args

	// Find positions of key arguments
	var modelPos, promptFlagPos, promptValuePos int = -1, -1, -1
	for i, arg := range args {
		switch arg {
		case "--model":
			modelPos = i
		case "-p":
			promptFlagPos = i
			if i+1 < len(args) {
				promptValuePos = i + 1
			}
		}
	}

	// Model should appear before prompt
	if modelPos != -1 && promptFlagPos != -1 {
		if modelPos > promptFlagPos {
			t.Error("--model should appear before -p flag")
		}
	}

	// Prompt value should follow prompt flag
	if promptFlagPos != -1 && promptValuePos != promptFlagPos+1 {
		t.Error("-p flag should be immediately followed by prompt value")
	}

	// Verify the prompt value
	if promptValuePos != -1 && args[promptValuePos] != "test prompt" {
		t.Errorf("prompt value = %q, want %q", args[promptValuePos], "test prompt")
	}
}

func TestNewClient_FlagsPresent(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test", "")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	requiredFlags := []string{
		"--dangerously-skip-permissions",
		"--print",
		"--verbose",
		"--output-format",
	}

	args := client.cmd.Args
	for _, flag := range requiredFlags {
		found := false
		for _, arg := range args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("NewClient() missing required flag: %s", flag)
		}
	}
}

func TestNewClient_OutputFormatValue(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test", "")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	args := client.cmd.Args
	for i, arg := range args {
		if arg == "--output-format" {
			if i+1 >= len(args) {
				t.Error("--output-format flag has no value")
				return
			}
			if args[i+1] != "stream-json" {
				t.Errorf("--output-format value = %q, want %q", args[i+1], "stream-json")
			}
			return
		}
	}
	t.Error("--output-format flag not found")
}
