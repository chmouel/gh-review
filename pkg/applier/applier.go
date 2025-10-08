package applier

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chmouel/gh-review/pkg/diffhunk"
	"github.com/chmouel/gh-review/pkg/github"
	"github.com/chmouel/gh-review/pkg/ui"
)

type Applier struct {
	debug bool
}

func New() *Applier {
	return &Applier{}
}

// SetDebug enables or disables debug output
func (a *Applier) SetDebug(debug bool) {
	a.debug = debug
}

// debugLog prints debug messages if debug mode is enabled
func (a *Applier) debugLog(format string, args ...interface{}) {
	if a.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
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

		// Show header with outdated warning if applicable
		header := fmt.Sprintf("[%d/%d] %s by @%s", i+1, len(suggestions), clickableLocation, suggestion.Author)
		if suggestion.IsOutdated {
			header += ui.Colorize(ui.ColorYellow, " ⚠️  OUTDATED")
		}
		fmt.Printf("\n%s\n", ui.Colorize(ui.ColorCyan, header))
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
		a.debugLog("Failed to create patch: %v", err)
		if a.debug {
			a.debugLog("Suggestion would have replaced with:\n%s", comment.SuggestedCode)
		}
		return fmt.Errorf("failed to create patch: %w", err)
	}

	a.debugLog("Generated patch:\n%s", patch)

	// Apply the patch using git apply
	cmd := exec.Command("git", "apply", "--unidiff-zero", "-")
	cmd.Stdin = strings.NewReader(patch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.debugLog("git apply failed: %v\nOutput: %s", err, string(output))
		// Save the patch to /tmp/ for manual inspection
		patchFile := fmt.Sprintf("/tmp/gh-review-patch-%d.patch", comment.ID)

		// Add diagnostic information to the patch file
		var patchWithInfo strings.Builder
		patchWithInfo.WriteString(fmt.Sprintf("# Failed to apply patch for comment ID %d\n", comment.ID))
		patchWithInfo.WriteString(fmt.Sprintf("# File: %s\n", comment.Path))
		patchWithInfo.WriteString(fmt.Sprintf("# Comment URL: %s\n", comment.HTMLURL))
		patchWithInfo.WriteString(fmt.Sprintf("# Error: %v\n", err))
		patchWithInfo.WriteString("# git apply output:\n")
		for _, line := range strings.Split(string(output), "\n") {
			patchWithInfo.WriteString(fmt.Sprintf("# %s\n", line))
		}
		patchWithInfo.WriteString("#\n# Generated patch:\n#\n")
		patchWithInfo.WriteString(patch)

		if writeErr := os.WriteFile(patchFile, []byte(patchWithInfo.String()), 0o644); writeErr == nil {
			return fmt.Errorf("failed to apply patch (saved to %s for manual review):\n%s", patchFile, string(output))
		}
		return fmt.Errorf("failed to apply patch: %w\nOutput: %s", err, string(output))
	}

	a.debugLog("Patch applied successfully!")
	return nil
}

// createPatch creates a unified diff patch from a GitHub suggestion
// This uses position mapping and diff hunk parsing for accurate line placement
func (a *Applier) createPatch(comment *github.ReviewComment) (string, error) {
	a.debugLog("Creating patch for comment ID=%d, Path=%s, Line=%d", comment.ID, comment.Path, comment.Line)
	a.debugLog("Comment position info: Line=%d, OriginalLine=%d, StartLine=%d, EndLine=%d",
		comment.Line, comment.OriginalLine, comment.StartLine, comment.EndLine)
	a.debugLog("DiffSide=%s, IsOutdated=%v", comment.DiffSide, comment.IsOutdated)

	// Read the current file
	fileContent, err := os.ReadFile(comment.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", comment.Path, err)
	}
	fileLines := strings.Split(string(fileContent), "\n")
	a.debugLog("Current file has %d lines", len(fileLines))

	// Extract the lines that were added in the PR (+ lines) from DiffHunk
	addedLines := diffhunk.GetAddedLines(comment.DiffHunk)
	a.debugLog("DiffHunk:\n%s", comment.DiffHunk)
	a.debugLog("Extracted %d added lines from diff hunk:", len(addedLines))
	for i, line := range addedLines {
		a.debugLog("  [%d] %q", i, line)
	}

	if len(addedLines) == 0 {
		return "", fmt.Errorf("no added lines found in diff hunk")
	}

	// Strategy 1: Try using position mapping from the diff hunk
	targetLine := -1

	if comment.DiffHunk != "" {
		// Parse the diff hunk to understand the structure
		parsedHunk, parseErr := diffhunk.ParseDiffHunk(comment.DiffHunk)
		if parseErr == nil {
			a.debugLog("Parsed diff hunk: OldStart=%d, OldLines=%d, NewStart=%d, NewLines=%d",
				parsedHunk.OldStart, parsedHunk.OldLines, parsedHunk.NewStart, parsedHunk.NewLines)

			// Use the first added line's position
			for _, line := range parsedHunk.Lines {
				if line.Type == diffhunk.Add {
					// Map from new file position to current file (0-based)
					targetLine = diffhunk.GetZeroBased(line.NewLineNumber)
					a.debugLog("Strategy 1 (position mapping): Found first added line at new position %d (0-based: %d)",
						line.NewLineNumber, targetLine)
					break
				}
			}
		} else {
			a.debugLog("Failed to parse diff hunk: %v", parseErr)
		}
	}

	// Strategy 2: Fall back to content matching if position mapping didn't work
	if targetLine == -1 {
		a.debugLog("Strategy 1 failed, trying Strategy 2 (content matching)")
		matchStart := -1
		for i := 0; i <= len(fileLines)-len(addedLines); i++ {
			match := true
			for j := 0; j < len(addedLines); j++ {
				if fileLines[i+j] != addedLines[j] {
					match = false
					break
				}
			}
			if match {
				matchStart = i
				a.debugLog("Strategy 2: Found content match at line %d (0-based)", matchStart)
				break
			}
		}

		if matchStart == -1 {
			a.debugLog("Strategy 2 failed: could not find matching content")
			return "", fmt.Errorf("could not find the code to replace in current file (looking for %d lines starting with %q)",
				len(addedLines), addedLines[0])
		}
		targetLine = matchStart
	}

	a.debugLog("Target line for replacement: %d (0-based), which is line %d (1-based)", targetLine, targetLine+1)

	// Verify the content matches at the target position
	if targetLine >= 0 && targetLine+len(addedLines) <= len(fileLines) {
		a.debugLog("Verifying content at target position...")
		a.debugLog("Current file content at target position:")
		for j := 0; j < len(addedLines) && targetLine+j < len(fileLines); j++ {
			a.debugLog("  [%d] Current: %q", targetLine+j+1, fileLines[targetLine+j])
			a.debugLog("  [%d] Expected: %q", targetLine+j+1, addedLines[j])
		}

		mismatch := false
		var mismatchLine int
		for j := 0; j < len(addedLines); j++ {
			if fileLines[targetLine+j] != addedLines[j] {
				mismatch = true
				mismatchLine = targetLine + j + 1
				a.debugLog("MISMATCH at line %d: got %q, expected %q",
					mismatchLine, fileLines[targetLine+j], addedLines[j])
				break
			}
		}
		if mismatch {
			// Show surrounding context
			a.debugLog("Showing file context around mismatch:")
			contextStart := targetLine - 3
			if contextStart < 0 {
				contextStart = 0
			}
			contextEnd := targetLine + len(addedLines) + 3
			if contextEnd > len(fileLines) {
				contextEnd = len(fileLines)
			}
			for i := contextStart; i < contextEnd; i++ {
				marker := "  "
				if i+1 == mismatchLine {
					marker = "→ "
				}
				a.debugLog("%s[%d] %q", marker, i+1, fileLines[i])
			}

			// Generate a diagnostic diff file showing the mismatch
			diffFile := a.saveMismatchDiff(comment, fileLines, targetLine, addedLines, mismatchLine)
			if diffFile != "" {
				return "", fmt.Errorf("content mismatch at line %d - the code may have changed since the review\nDiagnostic diff saved to: %s", mismatchLine, diffFile)
			}

			return "", fmt.Errorf("content mismatch at line %d - the code may have changed since the review", mismatchLine)
		}
		a.debugLog("Content verification passed!")
	}

	removeCount := len(addedLines)

	// Get context lines (3 before and after)
	contextSize := 3
	startLine := targetLine - contextSize
	if startLine < 0 {
		startLine = 0
	}
	endLine := targetLine + removeCount + contextSize
	if endLine > len(fileLines) {
		endLine = len(fileLines)
	}

	var patch strings.Builder

	// Write patch header
	patch.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", comment.Path, comment.Path))
	patch.WriteString(fmt.Sprintf("--- a/%s\n", comment.Path))
	patch.WriteString(fmt.Sprintf("+++ b/%s\n", comment.Path))

	// Count lines in the hunk
	oldLineCount := endLine - startLine
	suggestionLines := strings.Split(strings.TrimSuffix(comment.SuggestedCode, "\n"), "\n")
	newLineCount := (endLine - startLine) - removeCount + len(suggestionLines)

	// Write hunk header (1-based line numbers)
	patch.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startLine+1, oldLineCount, startLine+1, newLineCount))

	// Write context before
	for i := startLine; i < targetLine; i++ {
		patch.WriteString(" " + fileLines[i] + "\n")
	}

	// Write lines to remove
	for i := targetLine; i < targetLine+removeCount; i++ {
		patch.WriteString("-" + fileLines[i] + "\n")
	}

	// Write suggested lines
	for _, line := range suggestionLines {
		patch.WriteString("+" + line + "\n")
	}

	// Write context after
	for i := targetLine + removeCount; i < endLine; i++ {
		patch.WriteString(" " + fileLines[i] + "\n")
	}

	return patch.String(), nil
}

// saveMismatchDiff creates a diagnostic diff file showing what was expected vs what was found
func (a *Applier) saveMismatchDiff(comment *github.ReviewComment, fileLines []string, targetLine int, expectedLines []string, mismatchLine int) string {
	diffFile := fmt.Sprintf("/tmp/gh-review-mismatch-%d.diff", comment.ID)

	var diff strings.Builder

	// Header
	diff.WriteString(fmt.Sprintf("# Diagnostic diff for comment ID %d\n", comment.ID))
	diff.WriteString(fmt.Sprintf("# File: %s\n", comment.Path))
	diff.WriteString(fmt.Sprintf("# Comment URL: %s\n", comment.HTMLURL))
	diff.WriteString(fmt.Sprintf("# Mismatch at line: %d\n", mismatchLine))
	diff.WriteString(fmt.Sprintf("# Comment info: Line=%d, OriginalLine=%d, DiffSide=%s, IsOutdated=%v\n",
		comment.Line, comment.OriginalLine, comment.DiffSide, comment.IsOutdated))
	diff.WriteString("#\n")
	diff.WriteString("# Original diff hunk from GitHub:\n")
	for _, line := range strings.Split(comment.DiffHunk, "\n") {
		diff.WriteString(fmt.Sprintf("# %s\n", line))
	}
	diff.WriteString("#\n")
	diff.WriteString("# EXPECTED (from GitHub review):\n")
	for i, line := range expectedLines {
		marker := " "
		if targetLine+i+1 == mismatchLine {
			marker = "!"
		}
		diff.WriteString(fmt.Sprintf("# %s [%d] %s\n", marker, targetLine+i+1, line))
	}
	diff.WriteString("#\n")
	diff.WriteString("# ACTUAL (current file content):\n")
	for i := 0; i < len(expectedLines) && targetLine+i < len(fileLines); i++ {
		marker := " "
		if targetLine+i+1 == mismatchLine {
			marker = "!"
		}
		diff.WriteString(fmt.Sprintf("# %s [%d] %s\n", marker, targetLine+i+1, fileLines[targetLine+i]))
	}
	diff.WriteString("#\n")
	diff.WriteString("# Unified diff (proper format):\n")
	diff.WriteString("#\n")

	contextStart := targetLine - 5
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := targetLine + len(expectedLines) + 5
	if contextEnd > len(fileLines) {
		contextEnd = len(fileLines)
	}

	diff.WriteString(fmt.Sprintf("--- a/%s (expected based on review)\n", comment.Path))
	diff.WriteString(fmt.Sprintf("+++ b/%s (actual current content)\n", comment.Path))
	diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
		targetLine+1, len(expectedLines),
		targetLine+1, len(expectedLines)))

	// Show context before
	for i := contextStart; i < targetLine && i < len(fileLines); i++ {
		diff.WriteString(fmt.Sprintf(" %s\n", fileLines[i]))
	}

	// Show the expected lines (what review expected - as removed)
	for i := 0; i < len(expectedLines); i++ {
		diff.WriteString(fmt.Sprintf("-%s\n", expectedLines[i]))
	}

	// Show the actual lines (what we found - as added)
	for i := targetLine; i < targetLine+len(expectedLines) && i < len(fileLines); i++ {
		diff.WriteString(fmt.Sprintf("+%s\n", fileLines[i]))
	}

	// Show context after
	for i := targetLine + len(expectedLines); i < contextEnd && i < len(fileLines); i++ {
		diff.WriteString(fmt.Sprintf(" %s\n", fileLines[i]))
	}

	diff.WriteString("\n#\n")
	diff.WriteString("# Suggested change from review:\n")
	diff.WriteString("#\n")
	for _, line := range strings.Split(comment.SuggestedCode, "\n") {
		diff.WriteString(fmt.Sprintf("# > %s\n", line))
	}

	if err := os.WriteFile(diffFile, []byte(diff.String()), 0o644); err != nil {
		a.debugLog("Failed to save mismatch diff: %v", err)
		return ""
	}

	a.debugLog("Saved diagnostic diff to: %s", diffFile)
	return diffFile
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
