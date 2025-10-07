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
	var patch strings.Builder

	// Write patch header
	patch.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", comment.Path, comment.Path))
	patch.WriteString(fmt.Sprintf("--- a/%s\n", comment.Path))
	patch.WriteString(fmt.Sprintf("+++ b/%s\n", comment.Path))

	// Parse the diff hunk to extract the old lines and context
	hunkLines := strings.Split(comment.DiffHunk, "\n")

	// Find the starting line number from the hunk header (e.g., @@ -42,7 +42,7 @@)
	var oldStart, oldCount int
	if len(hunkLines) > 0 {
		// Parse hunk header like "@@ -42,7 +42,7 @@"
		hunkHeader := hunkLines[0]
		if strings.HasPrefix(hunkHeader, "@@") {
			fmt.Sscanf(hunkHeader, "@@ -%d,%d", &oldStart, &oldCount)
		}
	}

	// Count how many lines to remove from the original
	removeCount := 0
	for _, line := range hunkLines[1:] {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '-') {
			removeCount++
		}
	}

	// Count suggestion lines
	suggestionLines := strings.Split(comment.SuggestedCode, "\n")
	addCount := len(suggestionLines)

	// Write the hunk header
	// Use the line from the comment if we couldn't parse from hunk
	if oldStart == 0 {
		oldStart = comment.OriginalLine
		removeCount = comment.OriginalLines
	}

	patch.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, removeCount, oldStart, addCount))

	// Extract context and old lines from diff hunk
	contextBefore := []string{}
	contextAfter := []string{}
	oldLines := []string{}

	inOldCode := false
	for _, line := range hunkLines[1:] {
		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case ' ':
			// Context line
			if !inOldCode && len(oldLines) == 0 {
				contextBefore = append(contextBefore, line[1:])
			} else {
				contextAfter = append(contextAfter, line[1:])
			}
		case '-':
			// Old line to be removed
			inOldCode = true
			oldLines = append(oldLines, line[1:])
		case '+':
			// Skip GitHub's original suggested lines (we have our own)
			continue
		}
	}

	// Write context before
	for _, line := range contextBefore {
		patch.WriteString(" " + line + "\n")
	}

	// Write removed lines
	for _, line := range oldLines {
		patch.WriteString("-" + line + "\n")
	}

	// Write suggested lines (new content)
	for _, line := range suggestionLines {
		patch.WriteString("+" + line + "\n")
	}

	// Write context after
	for _, line := range contextAfter {
		patch.WriteString(" " + line + "\n")
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
