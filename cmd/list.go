package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chmouel/gh-review/pkg/github"
	"github.com/chmouel/gh-review/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	listShowResolved bool
	listDebug        bool
)

var listCmd = &cobra.Command{
	Use:   "list [PR_NUMBER]",
	Short: "List review comments for a pull request",
	Long:  `List all review comments and suggestions for a pull request.`,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&listShowResolved, "all", false, "Show resolved/done suggestions")
	listCmd.Flags().BoolVar(&listDebug, "debug", false, "Enable debug output")
}

func runList(cmd *cobra.Command, args []string) error {
	client := github.NewClient()
	client.SetDebug(listDebug)

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

	fmt.Printf("Found %d review comment(s):\n", len(filteredComments))

	for i, comment := range filteredComments {
		displayComment(i+1, len(filteredComments), comment)
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

// displayComment displays a single review comment with formatting
func displayComment(index, total int, comment *github.ReviewComment) {
	// Create clickable link to the review comment
	fileLocation := fmt.Sprintf("%s:%d", comment.Path, comment.Line)
	clickableLocation := ui.CreateHyperlink(comment.HTMLURL, fileLocation)

	// Header
	fmt.Printf("\n%s\n",
		ui.Colorize(ui.ColorCyan, fmt.Sprintf("[%d/%d] %s by @%s",
			index, total, clickableLocation, comment.Author)))
	fmt.Printf("%s\n", ui.Colorize(ui.ColorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	// Show resolved status
	if comment.IsResolved() {
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorGreen, "✅ Resolved"))
	}

	// Show the review comment (without the suggestion block)
	commentText := ui.StripSuggestionBlock(comment.Body)
	if commentText != "" {
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorYellow, "Review comment:"))
		rendered, err := ui.RenderMarkdown(commentText)
		if err == nil && rendered != "" {
			fmt.Println(rendered)
		} else {
			// Fallback to wrapped text
			wrappedComment := ui.WrapText(commentText, 80)
			fmt.Printf("%s\n", wrappedComment)
		}
	}

	// Show the suggestion if present
	if comment.HasSuggestion {
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorYellow, "Suggested change:"))
		fmt.Println(ui.ColorizeCode(comment.SuggestedCode))
	}

	// Show context (diff hunk) if available
	if comment.DiffHunk != "" {
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorYellow, "Context:"))
		fmt.Println(ui.ColorizeDiff(comment.DiffHunk))
	}

	// Show thread comments (replies)
	if len(comment.ThreadComments) > 0 {
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorCyan, "Thread replies:"))
		for i, threadComment := range comment.ThreadComments {
			fmt.Printf("\n  %s\n", ui.Colorize(ui.ColorGray, fmt.Sprintf("└─ Reply %d by @%s:", i+1, threadComment.Author)))
			rendered, err := ui.RenderMarkdown(threadComment.Body)
			if err == nil && rendered != "" {
				// Indent the rendered markdown
				lines := strings.Split(rendered, "\n")
				for _, line := range lines {
					fmt.Printf("     %s\n", line)
				}
			} else {
				// Fallback to wrapped text
				wrappedReply := ui.WrapText(threadComment.Body, 75)
				lines := strings.Split(wrappedReply, "\n")
				for _, line := range lines {
					fmt.Printf("     %s\n", line)
				}
			}
		}
	}

	fmt.Println()
}
