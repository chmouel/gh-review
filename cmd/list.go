package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/chmouel/gh-review/pkg/github"
	"github.com/spf13/cobra"
)

var (
	listShowResolved bool
)

var listCmd = &cobra.Command{
	Use:   "list [PR_NUMBER]",
	Short: "List review comments for a pull request",
	Long:  `List all review comments and suggestions for a pull request.`,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&listShowResolved, "all", false, "Show resolved/done suggestions")
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

	// Filter out resolved comments unless --all is specified
	filteredComments := make([]*github.ReviewComment, 0)
	for _, comment := range comments {
		if listShowResolved || !comment.IsResolved() {
			filteredComments = append(filteredComments, comment)
		}
	}

	if len(filteredComments) == 0 {
		if listShowResolved {
			fmt.Println("No review comments found.")
		} else {
			fmt.Println("No unresolved review comments found. Use --all to show resolved comments.")
		}
		return nil
	}

	fmt.Printf("Found %d review comment(s):\n\n", len(filteredComments))
	for i, comment := range filteredComments {
		fileLocation := fmt.Sprintf("%s:%d", comment.Path, comment.Line)
		clickableLocation := createHyperlink(comment.HTMLURL, fileLocation)

		fmt.Printf("[%d] %s\n", i+1, clickableLocation)
		fmt.Printf("    Author: %s\n", comment.Author)
		if comment.HasSuggestion {
			fmt.Printf("    ✨ Has suggestion\n")
		}
		if comment.IsResolved() {
			fmt.Printf("    ✅ Resolved\n")
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
		return 0, err
	}

	fmt.Fprintf(os.Stderr, "Auto-detected PR #%d for current branch\n", prNumber)
	return prNumber, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// createHyperlink creates an OSC8 hyperlink
func createHyperlink(url, text string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
