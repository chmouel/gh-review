package github

import (
	"encoding/json"
	"fmt"
	"os"
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
	SubjectType    string
	HTMLURL        string
}

// IsResolved returns true if the comment thread has been marked as resolved/done
func (rc *ReviewComment) IsResolved() bool {
	return rc.SubjectType == "resolved"
}

func NewClient() *Client {
	return &Client{}
}

// getResolvedThreads fetches resolved review thread IDs using GraphQL
func (c *Client) getResolvedThreads(repo string, prNumber int) (map[int64]bool, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s", repo)
	}
	owner := parts[0]
	name := parts[1]

	query := fmt.Sprintf(`
		query {
			repository(owner: "%s", name: "%s") {
				pullRequest(number: %d) {
					reviewThreads(first: 100) {
						nodes {
							id
							isResolved
							comments(first: 1) {
								nodes {
									databaseId
								}
							}
						}
					}
				}
			}
		}
	`, owner, name, prNumber)

	stdOut, _, err := gh.Exec("api", "graphql", "-f", fmt.Sprintf("query=%s", query))
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							ID         string `json:"id"`
							IsResolved bool   `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									DatabaseID int64 `json:"databaseId"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(stdOut.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	resolved := make(map[int64]bool)
	for _, thread := range result.Data.Repository.PullRequest.ReviewThreads.Nodes {
		if thread.IsResolved && len(thread.Comments.Nodes) > 0 {
			// Mark the first comment in the thread as resolved
			commentID := thread.Comments.Nodes[0].DatabaseID
			resolved[commentID] = true
		}
	}

	return resolved, nil
}

func (c *Client) getRepo() (string, error) {
	if c.repo != "" {
		return c.repo, nil
	}

	stdOut, _, err := gh.Exec("repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("not in a GitHub repository (or no remote configured)")
	}

	c.repo = strings.TrimSpace(stdOut.String())
	return c.repo, nil
}

func (c *Client) GetCurrentBranchPR() (int, error) {
	stdOut, _, err := gh.Exec("pr", "view", "--json", "number", "--jq", ".number")
	if err != nil {
		return 0, fmt.Errorf("no PR found for current branch (use: gh review list <PR_NUMBER>)")
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

	// First, get resolved thread IDs using GraphQL
	resolvedThreads, err := c.getResolvedThreads(repo, prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch resolved threads: %v\n", err)
		resolvedThreads = make(map[int64]bool)
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
		HTMLURL  string `json:"html_url"`
		User     struct {
			Login string `json:"login"`
		} `json:"user"`
		OriginalLine      int    `json:"original_line"`
		OriginalStartLine int    `json:"original_start_line"`
		SubjectType       string `json:"subject_type"`
	}

	if err := json.Unmarshal(stdOut.Bytes(), &rawComments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	comments := make([]*ReviewComment, 0, len(rawComments))
	for _, raw := range rawComments {
		// Check if this comment's thread is resolved
		subjectType := raw.SubjectType
		if resolvedThreads[raw.ID] {
			subjectType = "resolved"
		}

		comment := &ReviewComment{
			ID:          raw.ID,
			Path:        raw.Path,
			Line:        raw.Line,
			Body:        raw.Body,
			Author:      raw.User.Login,
			DiffHunk:    raw.DiffHunk,
			OriginalLine: raw.OriginalLine,
			SubjectType: subjectType,
			HTMLURL:     raw.HTMLURL,
		}

		// Debug: print full raw comment for troubleshooting
		if os.Getenv("GH_REVIEW_DEBUG") == "1" {
			debugJSON, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Fprintf(os.Stderr, "DEBUG: Comment %d (resolved=%v):\n%s\n", raw.ID, resolvedThreads[raw.ID], string(debugJSON))
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
