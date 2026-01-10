package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Issue represents a GitHub issue with its metadata
type Issue struct {
	Number    int
	Title     string
	Body      string
	Labels    []string
	URL       string
	Assignees []string
	Milestone string
}

// ParseIssueRef parses an issue reference into owner, repo, and issue number
// Supported formats:
//   - "42" or "#42" (issue number only, requires repo context)
//   - "owner/repo#42" (short reference)
//   - "https://github.com/owner/repo/issues/42" (full URL)
func ParseIssueRef(ref string) (owner, repo string, number int, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", 0, fmt.Errorf("empty issue reference")
	}

	// Try full URL format: https://github.com/owner/repo/issues/42
	urlPattern := regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/issues/(\d+)$`)
	if matches := urlPattern.FindStringSubmatch(ref); matches != nil {
		number, _ = strconv.Atoi(matches[3])
		return matches[1], matches[2], number, nil
	}

	// Try short reference format: owner/repo#42
	shortPattern := regexp.MustCompile(`^([^/]+)/([^#]+)#(\d+)$`)
	if matches := shortPattern.FindStringSubmatch(ref); matches != nil {
		number, _ = strconv.Atoi(matches[3])
		return matches[1], matches[2], number, nil
	}

	// Try issue number only: 42 or #42
	numberPattern := regexp.MustCompile(`^#?(\d+)$`)
	if matches := numberPattern.FindStringSubmatch(ref); matches != nil {
		number, _ = strconv.Atoi(matches[1])
		return "", "", number, nil
	}

	return "", "", 0, fmt.Errorf("invalid issue reference format: %s", ref)
}

// ghIssueResponse represents the JSON response from gh issue view
type ghIssueResponse struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []label  `json:"labels"`
	URL       string   `json:"url"`
	Assignees []user   `json:"assignees"`
	Milestone *msInfo  `json:"milestone"`
}

type label struct {
	Name string `json:"name"`
}

type user struct {
	Login string `json:"login"`
}

type msInfo struct {
	Title string `json:"title"`
}

// FetchIssue fetches an issue from GitHub using the gh CLI
// If owner/repo are empty, uses the current repository context
func FetchIssue(owner, repo string, number int) (*Issue, error) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found: please install GitHub CLI (https://cli.github.com)")
	}

	// Build the gh issue view command
	args := []string{"issue", "view", strconv.Itoa(number), "--json", "number,title,body,labels,url,assignees,milestone"}

	// Add repo flag if owner/repo are specified
	if owner != "" && repo != "" {
		args = append(args, "--repo", fmt.Sprintf("%s/%s", owner, repo))
	}

	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "Could not resolve to an issue") {
			if owner != "" && repo != "" {
				return nil, fmt.Errorf("issue #%d not found in %s/%s", number, owner, repo)
			}
			return nil, fmt.Errorf("issue #%d not found", number)
		}
		if strings.Contains(errMsg, "not a git repository") || strings.Contains(errMsg, "Could not resolve") {
			return nil, fmt.Errorf("cannot determine repository context: use full URL or run from a git repo")
		}
		return nil, fmt.Errorf("failed to fetch issue: %s", errMsg)
	}

	// Parse the JSON response
	var ghResp ghIssueResponse
	if err := json.Unmarshal(stdout.Bytes(), &ghResp); err != nil {
		return nil, fmt.Errorf("failed to parse issue response: %w", err)
	}

	// Convert to our Issue type
	issue := &Issue{
		Number: ghResp.Number,
		Title:  ghResp.Title,
		Body:   ghResp.Body,
		URL:    ghResp.URL,
	}

	// Extract label names
	for _, l := range ghResp.Labels {
		issue.Labels = append(issue.Labels, l.Name)
	}

	// Extract assignee logins
	for _, a := range ghResp.Assignees {
		issue.Assignees = append(issue.Assignees, a.Login)
	}

	// Extract milestone title
	if ghResp.Milestone != nil {
		issue.Milestone = ghResp.Milestone.Title
	}

	return issue, nil
}

// FetchIssueFromRef parses a reference and fetches the issue
// Convenience function that combines ParseIssueRef and FetchIssue
func FetchIssueFromRef(ref string) (*Issue, error) {
	owner, repo, number, err := ParseIssueRef(ref)
	if err != nil {
		return nil, err
	}
	return FetchIssue(owner, repo, number)
}
