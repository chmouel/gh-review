package parser

import (
	"testing"
)

func TestParseSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name: "simple suggestion",
			body: "You should change this:\n```suggestion\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			expected: "func main() {\n    fmt.Println(\"Hello\")\n}",
		},
		{
			name: "suggestion with context",
			body: "I think this would be better:\n\n```suggestion\nconst maxRetries = 3\n```\n\nWhat do you think?",
			expected: "const maxRetries = 3",
		},
		{
			name:     "no suggestion",
			body:     "This is just a regular comment without any code suggestions.",
			expected: "",
		},
		{
			name:     "code block but not suggestion",
			body:     "Here's an example:\n```go\nfmt.Println(\"test\")\n```",
			expected: "",
		},
		{
			name: "multiline suggestion",
			body: "```suggestion\nif err != nil {\n    return fmt.Errorf(\"failed: %w\", err)\n}\n```",
			expected: "if err != nil {\n    return fmt.Errorf(\"failed: %w\", err)\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSuggestion(tt.body)
			if result != tt.expected {
				t.Errorf("ParseSuggestion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseMultipleSuggestions(t *testing.T) {
	body := `You have two options:

Option 1:
` + "```suggestion\nconst timeout = 30\n```" + `

Option 2:
` + "```suggestion\nconst timeout = 60\n```"

	suggestions := ParseMultipleSuggestions(body)

	if len(suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(suggestions))
	}

	if suggestions[0] != "const timeout = 30" {
		t.Errorf("First suggestion = %q, want %q", suggestions[0], "const timeout = 30")
	}

	if suggestions[1] != "const timeout = 60" {
		t.Errorf("Second suggestion = %q, want %q", suggestions[1], "const timeout = 60")
	}
}
