package claude

// Runner defines the interface for running Claude processes.
// This interface enables mocking the Claude client for testing.
type Runner interface {
	// Start begins the Claude process
	Start() error

	// Wait waits for the process to complete and returns the exit code
	Wait() (int, error)

	// StreamOutput returns a channel of lines from stdout
	StreamOutput() <-chan string

	// Kill terminates the Claude process
	Kill() error
}

// Ensure Client implements Runner
var _ Runner = (*Client)(nil)
