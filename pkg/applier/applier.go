package applier

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chmouel/gh-review/pkg/github"
	"github.com/chmouel/gh-review/pkg/ui"
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

			// Show git diff of what was applied
			a.showGitDiff(suggestion.Path)
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
		// Create clickable link to the review comment
		fileLocation := fmt.Sprintf("%s:%d", suggestion.Path, suggestion.Line)
		clickableLocation := ui.CreateHyperlink(suggestion.HTMLURL, fileLocation)

		fmt.Printf("\n%s\n",
			ui.Colorize(ui.ColorCyan, fmt.Sprintf("[%d/%d] %s by @%s",
				i+1, len(suggestions), clickableLocation, suggestion.Author)))
		fmt.Printf("%s\n", ui.Colorize(ui.ColorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

		// Show the review comment (without the suggestion block)
		if commentText := ui.StripSuggestionBlock(suggestion.Body); commentText != "" {
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

		// Show the suggestion
		fmt.Printf("\n%s\n", "Suggested change:")
		fmt.Println(ui.ColorizeCode(suggestion.SuggestedCode))

		// Show context if available
		if suggestion.DiffHunk != "" {
			fmt.Printf("\n%s\n", "Context:")
			fmt.Println(ui.ColorizeDiff(suggestion.DiffHunk))
		}

		// Show thread comments (replies)
		if len(suggestion.ThreadComments) > 0 {
			fmt.Printf("\n%s\n", ui.Colorize(ui.ColorCyan, "Thread replies:"))
			for i, threadComment := range suggestion.ThreadComments {
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

		fmt.Printf("\n%s ", "Apply this suggestion? [y/s/q] (yes/skip/quit)")

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

				// Show git diff of what was applied
				a.showGitDiff(suggestion.Path)
			}
		case "s", "skip", "n", "no", "":
			fmt.Printf("⏭️  Skipped\n")
			skipped++
		default:
			fmt.Printf("⏭️  Skipped (unrecognized input)\n")
			skipped++
		}
	}

	fmt.Printf("\n%s\n", ui.Colorize(ui.ColorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Printf("%s Applied %s, Skipped %s\n",
		ui.Colorize(ui.ColorCyan, "Summary:"),
		ui.Colorize(ui.ColorGreen, fmt.Sprintf("%d", applied)),
		ui.Colorize(ui.ColorYellow, fmt.Sprintf("%d", skipped)))
	return nil
}

// applySuggestion applies a single suggestion to a file using git apply
func (a *Applier) applySuggestion(comment *github.ReviewComment) error {
	// Create a unified diff patch
	patch, err := a.createPatch(comment)
	if err != nil {
		return fmt.Errorf("failed to create patch: %w", err)
	}

	// Apply the patch using git apply
	cmd := exec.Command("git", "apply", "--unidiff-zero", "-")
	cmd.Stdin = strings.NewReader(patch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Save the patch to /tmp/ for manual inspection
		patchFile := fmt.Sprintf("/tmp/gh-review-patch-%d.patch", comment.ID)
		if writeErr := os.WriteFile(patchFile, []byte(patch), 0644); writeErr == nil {
			return fmt.Errorf("failed to apply patch (saved to %s for manual review):\n%s", patchFile, string(output))
		}
		return fmt.Errorf("failed to apply patch: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// createPatch creates a unified diff patch from a GitHub suggestion
func (a *Applier) createPatch(comment *github.ReviewComment) (string, error) {
	// Read the current file to get actual context
	fileContent, err := os.ReadFile(comment.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", comment.Path, err)
	}

	lines := strings.Split(string(fileContent), "\n")

	// Use Line (not OriginalLine) since we're on the PR branch
	// Line numbers are 1-based in GitHub API
	targetLine := comment.Line - 1 // Convert to 0-based

	// For multi-line suggestions, calculate how many lines to replace
	// by looking at the diff hunk
	removeCount := comment.OriginalLines
	if removeCount == 0 {
		removeCount = 1 // At least replace 1 line
	}

	// Validate line numbers
	if targetLine < 0 || targetLine >= len(lines) {
		return "", fmt.Errorf("line %d is out of range (file has %d lines)", comment.Line, len(lines))
	}
	if targetLine+removeCount > len(lines) {
		removeCount = len(lines) - targetLine
	}

	// Get context lines (3 lines before and after)
	contextSize := 3
	startLine := targetLine - contextSize
	if startLine < 0 {
		startLine = 0
	}
	endLine := targetLine + removeCount + contextSize
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var patch strings.Builder

	// Write patch header
	patch.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", comment.Path, comment.Path))
	patch.WriteString(fmt.Sprintf("--- a/%s\n", comment.Path))
	patch.WriteString(fmt.Sprintf("+++ b/%s\n", comment.Path))

	// Count the actual lines in the hunk
	oldLineCount := endLine - startLine
	suggestionLines := strings.Split(strings.TrimSuffix(comment.SuggestedCode, "\n"), "\n")
	newLineCount := (endLine - startLine) - removeCount + len(suggestionLines)

	// Write the hunk header (using 1-based line numbers)
	patch.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startLine+1, oldLineCount, startLine+1, newLineCount))

	// Write context before the change
	for i := startLine; i < targetLine; i++ {
		patch.WriteString(" " + lines[i] + "\n")
	}

	// Write the lines to be removed
	for i := targetLine; i < targetLine+removeCount; i++ {
		patch.WriteString("-" + lines[i] + "\n")
	}

	// Write the suggested lines (new content)
	for _, line := range suggestionLines {
		patch.WriteString("+" + line + "\n")
	}

	// Write context after the change
	for i := targetLine + removeCount; i < endLine; i++ {
		patch.WriteString(" " + lines[i] + "\n")
	}

	return patch.String(), nil
}

// showGitDiff shows the git diff for a file after applying changes
func (a *Applier) showGitDiff(filePath string) {
	// Execute git diff with color
	cmd := exec.Command("git", "diff", "--color=always", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail, just skip showing diff
		return
	}

	if len(output) > 0 && strings.TrimSpace(string(output)) != "" {
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorCyan, "Applied changes:"))
		fmt.Print(string(output))
	}
}
