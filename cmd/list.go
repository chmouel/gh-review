package cmd

import (
	"fmt"
	"strconv"

	"github.com/chmouel/gh-review/pkg/github"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [PR_NUMBER]",
	Short: "List review comments for a pull request",
	Long:  `List all review comments and suggestions for a pull request.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	client := github.NewClient()

	prNumber, err := getPRNumber(args, client)
	if err != nil {
		return err
	}

	comments, err := client.FetchReviewComments(prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch review comments: %w", err)
	}

	if len(comments) == 0 {
		fmt.Println("No review comments found.")
		return nil
	}

	fmt.Printf("Found %d review comment(s):\n\n", len(comments))
	for i, comment := range comments {
		fmt.Printf("[%d] %s:%d\n", i+1, comment.Path, comment.Line)
		fmt.Printf("    Author: %s\n", comment.Author)
		if comment.HasSuggestion {
			fmt.Printf("    ✨ Has suggestion\n")
		}
		fmt.Printf("    %s\n\n", truncate(comment.Body, 100))
	}

	return nil
}

func getPRNumber(args []string, client *github.Client) (int, error) {
	if len(args) > 0 {
		prNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid PR number: %s", args[0])
		}
		return prNumber, nil
	}

	// Get PR number for current branch
	prNumber, err := client.GetCurrentBranchPR()
	if err != nil {
		return 0, fmt.Errorf("failed to get PR for current branch: %w", err)
	}

	return prNumber, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
