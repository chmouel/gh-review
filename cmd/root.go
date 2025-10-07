package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-review",
	Short: "Apply GitHub review comments directly to your code",
	Long: `gh-review is a GitHub CLI extension that allows you to fetch and apply
review comments and suggestions from pull requests directly to your local code.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(debugCmd)
}
