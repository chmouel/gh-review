package cmd

import (
	"fmt"
	"strconv"

	"github.com/chmouel/gh-review/pkg/github"
	"github.com/spf13/cobra"
)

var (
	resolveUnresolve bool
	resolveDebug     bool
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [PR_NUMBER] <COMMENT_ID>",
	Short: "Resolve or unresolve a review comment thread",
	Long:  `Mark a review comment thread as resolved or unresolved.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runResolve,
}

func init() {
	resolveCmd.Flags().BoolVar(&resolveUnresolve, "unresolve", false, "Mark the thread as unresolved instead of resolved")
	resolveCmd.Flags().BoolVar(&resolveDebug, "debug", false, "Enable debug output")
}

func runResolve(cmd *cobra.Command, args []string) error {
	client := github.NewClient()
	client.SetDebug(resolveDebug)
	if repoFlag != "" {
		client.SetRepo(repoFlag)
	}

	var prNumber int
	var commentID int64
	var err error

	// Parse arguments: either "COMMENT_ID" or "PR_NUMBER COMMENT_ID"
	if len(args) == 1 {
		// Only comment ID provided, get PR from current branch
		prNumber, err = client.GetCurrentBranchPR()
		if err != nil {
			return err
		}
		commentID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid comment ID: %s", args[0])
		}
	} else {
		// Both PR number and comment ID provided
		prNumber, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
		commentID, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid comment ID: %s", args[1])
		}
	}

	// Fetch review comments to find the thread ID
	comments, err := client.FetchReviewComments(prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch review comments: %w", err)
	}

	// Find the comment with the given ID
	var threadID string
	for _, comment := range comments {
		if comment.ID == commentID {
			threadID = comment.ThreadID
			break
		}
	}

	if threadID == "" {
		return fmt.Errorf("comment ID %d not found in PR #%d", commentID, prNumber)
	}

	// Resolve or unresolve the thread
	if resolveUnresolve {
		if err := client.UnresolveThread(threadID); err != nil {
			return fmt.Errorf("failed to unresolve thread: %w", err)
		}
		fmt.Printf("✓ Thread for comment %d marked as unresolved\n", commentID)
	} else {
		if err := client.ResolveThread(threadID); err != nil {
			return fmt.Errorf("failed to resolve thread: %w", err)
		}
		fmt.Printf("✓ Thread for comment %d marked as resolved\n", commentID)
	}

	return nil
}
