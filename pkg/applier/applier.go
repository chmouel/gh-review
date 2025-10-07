package applier

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chmouel/gh-review/pkg/github"
)

type Applier struct{}

func New() *Applier {
	return &Applier{}
}

// ApplyAll applies all suggestions without prompting
func (a *Applier) ApplyAll(suggestions []*github.ReviewComment) error {
	applied := 0
	failed := 0

	for _, suggestion := range suggestions {
		if err := a.applySuggestion(suggestion); err != nil {
			fmt.Printf("❌ Failed to apply suggestion for %s:%d: %v\n",
				suggestion.Path, suggestion.Line, err)
			failed++
		} else {
			fmt.Printf("✅ Applied suggestion to %s:%d\n",
				suggestion.Path, suggestion.Line)
			applied++
		}
	}

	fmt.Printf("\nApplied %d/%d suggestions (%d failed)\n", applied, len(suggestions), failed)
	return nil
}

// ApplyInteractive prompts the user for each suggestion
func (a *Applier) ApplyInteractive(suggestions []*github.ReviewComment) error {
	reader := bufio.NewReader(os.Stdin)
	applied := 0
	skipped := 0

	for i, suggestion := range suggestions {
		fmt.Printf("\n[%d/%d] %s:%d by @%s\n",
			i+1, len(suggestions), suggestion.Path, suggestion.Line, suggestion.Author)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		// Show the suggestion
		fmt.Printf("\nSuggested change:\n")
		fmt.Printf("```\n%s\n```\n", suggestion.SuggestedCode)

		// Show context if available
		if suggestion.DiffHunk != "" {
			fmt.Printf("\nContext:\n%s\n", suggestion.DiffHunk)
		}

		fmt.Printf("\nApply this suggestion? [y/N/q] ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "q", "quit":
			fmt.Printf("\nStopped. Applied %d/%d suggestions\n", applied, i)
			return nil
		case "y", "yes":
			if err := a.applySuggestion(suggestion); err != nil {
				fmt.Printf("❌ Failed to apply: %v\n", err)
			} else {
				fmt.Printf("✅ Applied\n")
				applied++
			}
		default:
			fmt.Printf("⏭️  Skipped\n")
			skipped++
		}
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Summary: Applied %d, Skipped %d\n", applied, skipped)
	return nil
}

// applySuggestion applies a single suggestion to a file
func (a *Applier) applySuggestion(comment *github.ReviewComment) error {
	// Read the file
	content, err := os.ReadFile(comment.Path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Validate line number
	if comment.OriginalLine < 1 || comment.OriginalLine > len(lines) {
		return fmt.Errorf("invalid line number: %d (file has %d lines)",
			comment.OriginalLine, len(lines))
	}

	// Calculate the range of lines to replace
	startLine := comment.OriginalLine - 1 // Convert to 0-indexed
	endLine := startLine + comment.OriginalLines

	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Split the suggestion into lines
	suggestionLines := strings.Split(comment.SuggestedCode, "\n")

	// Build the new content
	newLines := make([]string, 0, len(lines)-comment.OriginalLines+len(suggestionLines))
	newLines = append(newLines, lines[:startLine]...)
	newLines = append(newLines, suggestionLines...)
	newLines = append(newLines, lines[endLine:]...)

	// Write back to the file
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(comment.Path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
