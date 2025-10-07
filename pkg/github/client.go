package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/chmouel/gh-review/pkg/parser"
)

type Client struct {
	repo string
}

type ReviewComment struct {
	ID             int64
	Path           string
	Line           int
	Body           string
	Author         string
	HasSuggestion  bool
	SuggestedCode  string
	OriginalLine   int
	OriginalLines  int
	DiffHunk       string
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) getRepo() (string, error) {
	if c.repo != "" {
		return c.repo, nil
	}

	stdOut, _, err := gh.Exec("repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	c.repo = strings.TrimSpace(stdOut.String())
	return c.repo, nil
}

func (c *Client) GetCurrentBranchPR() (int, error) {
	repo, err := c.getRepo()
	if err != nil {
		return 0, err
	}

	stdOut, _, err := gh.Exec("pr", "view", "--repo", repo, "--json", "number", "-q", ".number")
	if err != nil {
		return 0, fmt.Errorf("failed to get PR for current branch: %w", err)
	}

	var prNumber int
	if err := json.Unmarshal(stdOut.Bytes(), &prNumber); err != nil {
		return 0, fmt.Errorf("failed to parse PR number: %w", err)
	}

	return prNumber, nil
}

func (c *Client) FetchReviewComments(prNumber int) ([]*ReviewComment, error) {
	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}

	// Fetch review comments using gh api
	query := fmt.Sprintf("repos/%s/pulls/%d/comments", repo, prNumber)
	stdOut, _, err := gh.Exec("api", query, "--paginate")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch review comments: %w", err)
	}

	var rawComments []struct {
		ID       int64  `json:"id"`
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Body     string `json:"body"`
		DiffHunk string `json:"diff_hunk"`
		User     struct {
			Login string `json:"login"`
		} `json:"user"`
		OriginalLine int `json:"original_line"`
		OriginalStartLine int `json:"original_start_line"`
	}

	if err := json.Unmarshal(stdOut.Bytes(), &rawComments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	comments := make([]*ReviewComment, 0, len(rawComments))
	for _, raw := range rawComments {
		comment := &ReviewComment{
			ID:       raw.ID,
			Path:     raw.Path,
			Line:     raw.Line,
			Body:     raw.Body,
			Author:   raw.User.Login,
			DiffHunk: raw.DiffHunk,
			OriginalLine: raw.OriginalLine,
		}

		// Check if the comment contains a suggestion
		if suggestion := parser.ParseSuggestion(raw.Body); suggestion != "" {
			comment.HasSuggestion = true
			comment.SuggestedCode = suggestion

			// Calculate how many lines the suggestion spans
			comment.OriginalLines = calculateOriginalLines(raw.DiffHunk, raw.OriginalLine)
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// calculateOriginalLines determines how many lines from the original file
// should be replaced based on the diff hunk
func calculateOriginalLines(diffHunk string, originalLine int) int {
	lines := strings.Split(diffHunk, "\n")
	count := 0

	for _, line := range lines {
		// Count lines that start with ' ' or '-' (context or removed lines)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '-') {
			count++
		}
	}

	// Default to 1 if we can't determine
	if count == 0 {
		return 1
	}

	return count
}
