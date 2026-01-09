package claude

import (
	"bufio"
	"context"
	"io"
	"os/exec"
)

// Client manages Claude process execution
type Client struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// NewClient creates a new Claude client
// model is optional - if empty, uses the default model
func NewClient(ctx context.Context, prompt string, model string) (*Client, error) {
	args := []string{
		"--dangerously-skip-permissions",
		"--print",
		"--verbose",
		"--output-format", "stream-json",
	}

	if model != "" {
		args = append(args, "--model", model)
	}

	args = append(args, "-p", prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return nil, err
	}

	return &Client{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// Start begins the Claude process
func (c *Client) Start() error {
	return c.cmd.Start()
}

// Wait waits for the process to complete and returns the exit code
func (c *Client) Wait() (int, error) {
	err := c.cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), err
		}
		return -1, err
	}
	return 0, nil
}

// StreamOutput returns a channel of lines from both stdout and stderr
func (c *Client) StreamOutput() <-chan string {
	lines := make(chan string, 100)

	go func() {
		defer close(lines)

		// Read stdout
		scanner := bufio.NewScanner(c.stdout)
		// Increase buffer size for potentially large JSON lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024) // 1MB max line size

		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	// Also consume stderr but don't send it (it gets mixed with stdout in bash version)
	go func() {
		io.Copy(io.Discard, c.stderr)
	}()

	return lines
}

// Kill terminates the Claude process
func (c *Client) Kill() error {
	if c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}
