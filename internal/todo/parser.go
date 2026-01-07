package todo

import (
	"bufio"
	"os"
	"regexp"
)

var (
	pendingPattern   = regexp.MustCompile(`^-\s*\[\s*\]`)
	completedPattern = regexp.MustCompile(`^-\s*\[[xX]\]`)
)

// Counts holds todo item counts
type Counts struct {
	Pending   int
	Completed int
}

// ParseFile parses a TODO.md file and returns counts
func ParseFile(path string) (*Counts, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No TODO.md file yet, return zero counts
			return &Counts{}, nil
		}
		return nil, err
	}
	defer file.Close()

	counts := &Counts{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if pendingPattern.MatchString(line) {
			counts.Pending++
		} else if completedPattern.MatchString(line) {
			counts.Completed++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

// Total returns the total number of todo items
func (c *Counts) Total() int {
	return c.Pending + c.Completed
}

// CompletionRate returns the completion rate as a percentage
func (c *Counts) CompletionRate() float64 {
	total := c.Total()
	if total == 0 {
		return 0
	}
	return float64(c.Completed) / float64(total) * 100
}
