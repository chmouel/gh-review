package cmd

import (
	"fmt"

	"github.com/chmouel/gh-review/pkg/applier"
	"github.com/chmouel/gh-review/pkg/github"
	"github.com/spf13/cobra"
)

var (
	applyAll  bool
	applyFile string
)

var applyCmd = &cobra.Command{
	Use:   "apply [PR_NUMBER]",
	Short: "Apply review suggestions to local files",
	Long:  `Apply GitHub review suggestions to your local files interactively or in batch mode.`,
	RunE:  runApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyAll, "all", false, "Apply all suggestions without prompting")
	applyCmd.Flags().StringVar(&applyFile, "file", "", "Only apply suggestions for a specific file")
}

func runApply(cmd *cobra.Command, args []string) error {
	client := github.NewClient()

	prNumber, err := getPRNumber(args, client)
	if err != nil {
		return err
	}

	comments, err := client.FetchReviewComments(prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch review comments: %w", err)
	}

	// Filter comments with suggestions
	suggestions := make([]*github.ReviewComment, 0)
	for _, comment := range comments {
		if comment.HasSuggestion {
			if applyFile == "" || comment.Path == applyFile {
				suggestions = append(suggestions, comment)
			}
		}
	}

	if len(suggestions) == 0 {
		if applyFile != "" {
			fmt.Printf("No suggestions found for file: %s\n", applyFile)
		} else {
			fmt.Println("No suggestions found in review comments.")
		}
		return nil
	}

	fmt.Printf("Found %d suggestion(s) to apply\n\n", len(suggestions))

	app := applier.New()

	if applyAll {
		return app.ApplyAll(suggestions)
	}

	return app.ApplyInteractive(suggestions)
}
