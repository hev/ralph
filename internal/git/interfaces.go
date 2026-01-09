package git

import "os/exec"

// CommandExecutor defines an interface for executing shell commands.
// This enables mocking git/gh CLI calls for testing.
type CommandExecutor interface {
	// Run executes a command and returns its output
	Run(name string, args ...string) ([]byte, error)
}

// PRCreator defines the interface for creating pull requests.
type PRCreator interface {
	// CreatePR creates a pull request with the given configuration
	CreatePR(cfg PRConfig) (*PRResult, error)
}

// CommitTracker defines the interface for tracking git commits.
type CommitTracker interface {
	// CommitsDelta returns the number of commits made since the tracker was created
	CommitsDelta() (int, error)

	// UpdateBaseline updates the initial commit count to the current count
	UpdateBaseline() error
}

// Ensure Tracker implements CommitTracker
var _ CommitTracker = (*Tracker)(nil)

// DefaultCommandExecutor is the default implementation using os/exec.
type DefaultCommandExecutor struct{}

// Run executes a command using os/exec.
func (e *DefaultCommandExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
