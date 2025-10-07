package applier

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chmouel/gh-review/pkg/github"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

type Applier struct{}

func New() *Applier {
	return &Applier{}
}

// colorize applies ANSI color codes to text
func colorize(color, text string) string {
	return color + text + colorReset
}

// colorizeDiff applies syntax highlighting to diff hunks
func colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var coloredLines []string

	for _, line := range lines {
		if len(line) == 0 {
			coloredLines = append(coloredLines, line)
			continue
		}

		switch line[0] {
		case '+':
			coloredLines = append(coloredLines, colorize(colorGreen, line))
		case '-':
			coloredLines = append(coloredLines, colorize(colorRed, line))
		case '@':
			coloredLines = append(coloredLines, colorize(colorCyan, line))
		default:
			coloredLines = append(coloredLines, colorize(colorGray, line))
		}
	}

	return strings.Join(coloredLines, "\n")
}

// colorizeCode applies syntax highlighting to suggested code
func colorizeCode(code string) string {
	return colorize(colorGreen, code)
}

// createHyperlink creates an OSC8 hyperlink
// Format: \033]8;;URL\033\\TEXT\033]8;;\033\\
func createHyperlink(url, text string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
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
		// Create clickable link to the review comment
		fileLocation := fmt.Sprintf("%s:%d", suggestion.Path, suggestion.Line)
		clickableLocation := createHyperlink(suggestion.HTMLURL, fileLocation)

		fmt.Printf("\n%s\n",
			colorize(colorCyan, fmt.Sprintf("[%d/%d] %s by @%s",
				i+1, len(suggestions), clickableLocation, suggestion.Author)))
		fmt.Printf("%s\n", colorize(colorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

		// Show the suggestion
		fmt.Printf("\n%s\n", "Suggested change:")
		fmt.Println(colorizeCode(suggestion.SuggestedCode))

		// Show context if available
		if suggestion.DiffHunk != "" {
			fmt.Printf("\n%s\n", "Context:")
			fmt.Println(colorizeDiff(suggestion.DiffHunk))
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
			}
		case "s", "skip", "n", "no", "":
			fmt.Printf("⏭️  Skipped\n")
			skipped++
		default:
			fmt.Printf("⏭️  Skipped (unrecognized input)\n")
			skipped++
		}
	}

	fmt.Printf("\n%s\n", colorize(colorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Printf("%s Applied %s, Skipped %s\n",
		colorize(colorCyan, "Summary:"),
		colorize(colorGreen, fmt.Sprintf("%d", applied)),
		colorize(colorYellow, fmt.Sprintf("%d", skipped)))
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
