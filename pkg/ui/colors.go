package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Colorize applies ANSI color codes to text
func Colorize(color, text string) string {
	return color + text + ColorReset
}

// ColorizeDiff applies syntax highlighting to diff hunks
func ColorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var coloredLines []string

	for _, line := range lines {
		if len(line) == 0 {
			coloredLines = append(coloredLines, line)
			continue
		}

		switch line[0] {
		case '+':
			coloredLines = append(coloredLines, Colorize(ColorGreen, line))
		case '-':
			coloredLines = append(coloredLines, Colorize(ColorRed, line))
		case '@':
			coloredLines = append(coloredLines, Colorize(ColorCyan, line))
		default:
			coloredLines = append(coloredLines, Colorize(ColorGray, line))
		}
	}

	return strings.Join(coloredLines, "\n")
}

// ColorizeCode applies syntax highlighting to suggested code
func ColorizeCode(code string) string {
	return Colorize(ColorGreen, code)
}

// CreateHyperlink creates an OSC8 hyperlink
func CreateHyperlink(url, text string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// StripSuggestionBlock removes the suggestion code block and images from comment body
func StripSuggestionBlock(body string) string {
	result := strings.TrimSpace(body)

	// Remove ```suggestion...``` blocks
	suggestionRe := regexp.MustCompile("(?s)```suggestion\\s*\\n.*?```")
	result = suggestionRe.ReplaceAllString(result, "")

	// Remove markdown image links like ![alt](url)
	imageRe := regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	result = imageRe.ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

// WrapText wraps text to a maximum line width
func WrapText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine []string
	currentLength := 0

	for _, word := range words {
		wordLen := len(word)

		// If adding this word would exceed width, start a new line
		if currentLength > 0 && currentLength+1+wordLen > width {
			lines = append(lines, strings.Join(currentLine, " "))
			currentLine = []string{word}
			currentLength = wordLen
		} else {
			currentLine = append(currentLine, word)
			if currentLength > 0 {
				currentLength += 1 + wordLen // +1 for space
			} else {
				currentLength = wordLen
			}
		}
	}

	// Add the last line
	if len(currentLine) > 0 {
		lines = append(lines, strings.Join(currentLine, " "))
	}

	return strings.Join(lines, "\n")
}

// RenderMarkdown renders markdown text with glamour
func RenderMarkdown(text string) (string, error) {
	if text == "" {
		return "", nil
	}

	// Create a glamour renderer with dark style
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback to plain text if rendering fails
		return text, nil
	}

	rendered, err := r.Render(text)
	if err != nil {
		// Fallback to plain text if rendering fails
		return text, nil
	}

	return strings.TrimSpace(rendered), nil
}
