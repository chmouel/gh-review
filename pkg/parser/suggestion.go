package parser

import (
	"regexp"
	"strings"
)

// ParseSuggestion extracts the suggested code from a GitHub review comment body
// GitHub suggestions are in the format:
// ```suggestion
// suggested code here
// ```
func ParseSuggestion(body string) string {
	// Match code blocks with "suggestion" language
	re := regexp.MustCompile("(?s)```suggestion\\s*\\n(.*?)```")
	matches := re.FindStringSubmatch(body)

	if len(matches) < 2 {
		return ""
	}

	return strings.TrimRight(matches[1], "\n")
}

// ParseMultipleSuggestions extracts all suggestions from a comment body
func ParseMultipleSuggestions(body string) []string {
	re := regexp.MustCompile("(?s)```suggestion\\s*\\n(.*?)```")
	matches := re.FindAllStringSubmatch(body, -1)

	suggestions := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			suggestions = append(suggestions, strings.TrimRight(match[1], "\n"))
		}
	}

	return suggestions
}
