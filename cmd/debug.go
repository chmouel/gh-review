package cmd

import (
	"fmt"
	"strconv"

	"github.com/chmouel/gh-prreview/pkg/github"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug <PR_NUMBER> <COMMENT_ID>",
	Short: "Show debug information for a review comment",
	Long:  `Display the raw JSON data for a specific review comment (useful for debugging).`,
	Args:  cobra.ExactArgs(2),
	RunE:  runDebug,
}

func runDebug(cmd *cobra.Command, args []string) error {
	prNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}

	commentID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid comment ID: %w", err)
	}

	client := github.NewClient()
	client.SetDebug(true)
	if repoFlag != "" {
		client.SetRepo(repoFlag)
	}

	jsonData, err := client.DumpCommentJSON(prNumber, commentID)
	if err != nil {
		return fmt.Errorf("failed to dump comment: %w", err)
	}

	fmt.Println("Raw JSON for comment:")
	fmt.Println(jsonData)

	return nil
}
