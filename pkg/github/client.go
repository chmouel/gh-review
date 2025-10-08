package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/chmouel/gh-review/pkg/diffposition"
	"github.com/chmouel/gh-review/pkg/parser"
	"github.com/cli/go-gh/v2"
)

type Client struct {
	repo  string
	debug bool
}

type ReviewComment struct {
	ID                int64
	ThreadID          string // GraphQL node ID for resolving the thread
	Path              string
	Line              int
	Body              string
	Author            string
	HasSuggestion     bool
	SuggestedCode     string
	OriginalLine      int
	OriginalLines     int
	StartLine         int
	EndLine           int
	OriginalStartLine int
	OriginalEndLine   int
	DiffHunk          string
	DiffSide          diffposition.DiffSide
	SubjectType       string
	HTMLURL           string
	IsOutdated        bool
	ThreadComments    []ThreadComment
}

type ThreadComment struct {
	ID      int64
	Body    string
	Author  string
	HTMLURL string
}

// IsResolved returns true if the comment thread has been marked as resolved/done
func (rc *ReviewComment) IsResolved() bool {
	return rc.SubjectType == "resolved"
}

func NewClient() *Client {
	return &Client{}
}

// SetDebug enables or disables debug output
func (c *Client) SetDebug(debug bool) {
	c.debug = debug
}

// SetRepo sets the repository to use (format: "owner/repo")
func (c *Client) SetRepo(repo string) {
	c.repo = repo
}

// debugLog prints debug messages if debug mode is enabled
func (c *Client) debugLog(format string, args ...any) {
	if c.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// ThreadInfo contains information about a review thread
type ThreadInfo struct {
	ID         string // GraphQL node ID for resolving the thread
	IsResolved bool
	Comments   []ThreadComment
}

// getReviewThreads fetches review threads with all comments using GraphQL
func (c *Client) getReviewThreads(repo string, prNumber int) (map[int64]*ThreadInfo, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s", repo)
	}
	owner := parts[0]
	name := parts[1]

	c.debugLog("Fetching review threads for %s PR #%d", repo, prNumber)

	query := fmt.Sprintf(`
		query {
			repository(owner: "%s", name: "%s") {
				pullRequest(number: %d) {
					reviewThreads(first: 100) {
						nodes {
							id
							isResolved
							comments(first: 50) {
								nodes {
									databaseId
									body
									url
									author {
										login
									}
								}
							}
						}
					}
				}
			}
		}
	`, owner, name, prNumber)

	c.debugLog("GraphQL query: %s", query)

	stdOut, _, err := gh.Exec("api", "graphql", "-f", fmt.Sprintf("query=%s", query))
	if err != nil {
		c.debugLog("GraphQL query failed: %v", err)
		return nil, err
	}

	c.debugLog("GraphQL response length: %d bytes", len(stdOut.Bytes()))

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
									DatabaseID int64  `json:"databaseId"`
									Body       string `json:"body"`
									URL        string `json:"url"`
									Author     struct {
										Login string `json:"login"`
									} `json:"author"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(stdOut.Bytes(), &result); err != nil {
		c.debugLog("Failed to parse GraphQL response: %v", err)
		if c.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Raw response: %s\n", stdOut.String())
		}
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	c.debugLog("Found %d review threads", len(result.Data.Repository.PullRequest.ReviewThreads.Nodes))

	threads := make(map[int64]*ThreadInfo)
	for i, thread := range result.Data.Repository.PullRequest.ReviewThreads.Nodes {
		if len(thread.Comments.Nodes) == 0 {
			c.debugLog("Thread %d: no comments, skipping", i)
			continue
		}

		// First comment is the key
		firstCommentID := thread.Comments.Nodes[0].DatabaseID
		c.debugLog("Thread %d: first comment ID=%d, resolved=%v, comments=%d",
			i, firstCommentID, thread.IsResolved, len(thread.Comments.Nodes))

		// Collect all comments in the thread
		var threadComments []ThreadComment
		for j, comment := range thread.Comments.Nodes {
			c.debugLog("  Comment %d: ID=%d, author=%s, body_len=%d",
				j, comment.DatabaseID, comment.Author.Login, len(comment.Body))
			threadComments = append(threadComments, ThreadComment{
				ID:      comment.DatabaseID,
				Body:    comment.Body,
				Author:  comment.Author.Login,
				HTMLURL: comment.URL,
			})
		}

		threads[firstCommentID] = &ThreadInfo{
			ID:         thread.ID,
			IsResolved: thread.IsResolved,
			Comments:   threadComments,
		}
	}

	c.debugLog("Returning %d threads", len(threads))

	return threads, nil
}

// getReplyCommentIDs returns a set of comment IDs that are replies (not first comments in threads)
func (c *Client) getReplyCommentIDs(threads map[int64]*ThreadInfo) map[int64]bool {
	replyIDs := make(map[int64]bool)
	for firstCommentID, threadInfo := range threads {
		for _, comment := range threadInfo.Comments {
			if comment.ID != firstCommentID {
				replyIDs[comment.ID] = true
			}
		}
	}
	return replyIDs
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

// DumpCommentJSON returns the raw JSON for a specific comment (for debugging)
func (c *Client) DumpCommentJSON(prNumber int, commentID int64) (string, error) {
	repo, err := c.getRepo()
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("repos/%s/pulls/%d/comments", repo, prNumber)
	stdOut, _, err := gh.Exec("api", query, "--paginate")
	if err != nil {
		return "", fmt.Errorf("failed to fetch review comments: %w", err)
	}

	var rawComments []json.RawMessage
	if err := json.Unmarshal(stdOut.Bytes(), &rawComments); err != nil {
		return "", fmt.Errorf("failed to parse comments: %w", err)
	}

	for _, raw := range rawComments {
		var comment struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(raw, &comment); err != nil {
			continue
		}
		if comment.ID == commentID {
			// Pretty print the JSON
			var prettyJSON any
			_ = json.Unmarshal(raw, &prettyJSON)
			formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
			return string(formatted), nil
		}
	}

	return "", fmt.Errorf("comment ID %d not found", commentID)
}

func (c *Client) FetchReviewComments(prNumber int) ([]*ReviewComment, error) {
	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}

	// First, get review threads with all comments using GraphQL
	reviewThreads, err := c.getReviewThreads(repo, prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch review threads: %v\n", err)
		reviewThreads = make(map[int64]*ThreadInfo)
	}

	// Fetch review comments using gh api
	query := fmt.Sprintf("repos/%s/pulls/%d/comments", repo, prNumber)
	stdOut, _, err := gh.Exec("api", query, "--paginate")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch review comments: %w", err)
	}

	var rawComments []struct {
		ID        int64  `json:"id"`
		Path      string `json:"path"`
		Line      int    `json:"line"`
		StartLine int    `json:"start_line"`
		Body      string `json:"body"`
		DiffHunk  string `json:"diff_hunk"`
		HTMLURL   string `json:"html_url"`
		Side      string `json:"side"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		OriginalLine      int    `json:"original_line"`
		OriginalStartLine int    `json:"original_start_line"`
		SubjectType       string `json:"subject_type"`
	}

	if err := json.Unmarshal(stdOut.Bytes(), &rawComments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	c.debugLog("Processing %d review comments from REST API", len(rawComments))

	// Get set of reply comment IDs to skip
	replyIDs := c.getReplyCommentIDs(reviewThreads)

	comments := make([]*ReviewComment, 0, len(rawComments))
	for _, raw := range rawComments {
		// Skip reply comments - they're already in ThreadComments
		if replyIDs[raw.ID] {
			c.debugLog("Comment %d: Skipping (it's a reply, not a top-level review comment)", raw.ID)
			continue
		}

		// Check if this comment has thread info
		threadInfo := reviewThreads[raw.ID]
		subjectType := raw.SubjectType
		var threadComments []ThreadComment
		var threadID string

		if threadInfo != nil {
			c.debugLog("Comment %d: Found thread with %d total comments, resolved=%v",
				raw.ID, len(threadInfo.Comments), threadInfo.IsResolved)
			threadID = threadInfo.ID
			if threadInfo.IsResolved {
				subjectType = "resolved"
			}
			// Skip the first comment (it's the main review comment we're already showing)
			if len(threadInfo.Comments) > 1 {
				threadComments = threadInfo.Comments[1:]
				c.debugLog("Comment %d: Adding %d thread replies", raw.ID, len(threadComments))
			}
		} else {
			c.debugLog("Comment %d: No thread info found", raw.ID)
		}

		// Determine diff side
		diffSide := diffposition.DiffSideRight
		if raw.Side == "LEFT" {
			diffSide = diffposition.DiffSideLeft
		}

		// Calculate position information
		startLine := raw.Line
		if raw.StartLine > 0 {
			startLine = raw.StartLine
		}
		endLine := raw.Line

		originalStartLine := raw.OriginalLine
		if raw.OriginalStartLine > 0 {
			originalStartLine = raw.OriginalStartLine
		}
		originalEndLine := raw.OriginalLine

		// Calculate if comment is outdated
		isOutdated := false
		if raw.DiffHunk != "" {
			pos, err := diffposition.CalculateCommentPosition(
				raw.Line,
				raw.OriginalLine,
				raw.DiffHunk,
				diffSide,
			)
			if err == nil {
				isOutdated = pos.IsOutdated
			}
		}

		comment := &ReviewComment{
			ID:                raw.ID,
			ThreadID:          threadID,
			Path:              raw.Path,
			Line:              raw.Line,
			StartLine:         startLine,
			EndLine:           endLine,
			Body:              raw.Body,
			Author:            raw.User.Login,
			DiffHunk:          raw.DiffHunk,
			DiffSide:          diffSide,
			OriginalLine:      raw.OriginalLine,
			OriginalStartLine: originalStartLine,
			OriginalEndLine:   originalEndLine,
			SubjectType:       subjectType,
			HTMLURL:           raw.HTMLURL,
			IsOutdated:        isOutdated,
			ThreadComments:    threadComments,
		}

		// Check if the comment contains a suggestion
		if suggestion := parser.ParseSuggestion(raw.Body); suggestion != "" {
			comment.HasSuggestion = true
			comment.SuggestedCode = suggestion

			// Calculate how many lines the suggestion spans
			comment.OriginalLines = calculateOriginalLines(raw.DiffHunk)
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// calculateOriginalLines determines how many lines from the original file
// should be replaced based on the diff hunk
func calculateOriginalLines(diffHunk string) int {
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

// ResolveThread marks a review thread as resolved using GraphQL
func (c *Client) ResolveThread(threadID string) error {
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}

	c.debugLog("Resolving thread with ID: %s", threadID)

	mutation := `
		mutation ResolveThread($threadId: ID!) {
			resolveReviewThread(input: {threadId: $threadId}) {
				thread {
					id
					isResolved
				}
			}
		}
	`

	c.debugLog("GraphQL mutation: %s (threadId=%s)", mutation, threadID)

	stdOut, stdErr, err := gh.Exec("api", "graphql",
		"-f", fmt.Sprintf("query=%s", mutation),
		"-F", fmt.Sprintf("threadId=%s", threadID))
	if err != nil {
		c.debugLog("GraphQL mutation failed: %v", err)
		if stdErr.Len() > 0 {
			c.debugLog("Stderr: %s", stdErr.String())
		}
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	c.debugLog("GraphQL response length: %d bytes", len(stdOut.Bytes()))

	// Parse response to verify it worked
	var result struct {
		Data struct {
			ResolveReviewThread struct {
				Thread struct {
					ID         string `json:"id"`
					IsResolved bool   `json:"isResolved"`
				} `json:"thread"`
			} `json:"resolveReviewThread"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(stdOut.Bytes(), &result); err != nil {
		c.debugLog("Failed to parse GraphQL response: %v", err)
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	if !result.Data.ResolveReviewThread.Thread.IsResolved {
		return fmt.Errorf("thread was not marked as resolved")
	}

	c.debugLog("Thread resolved successfully")
	return nil
}
