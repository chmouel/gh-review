package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiProvider implements AIProvider for Google's Gemini models
type GeminiProvider struct {
	client         *genai.Client
	model          string
	templateConfig *TemplateConfig
}

// NewGeminiProvider creates a new Gemini AI provider
func NewGeminiProvider(apiKey string, model string, templateConfig *TemplateConfig) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	if model == "" {
		model = "gemini-1.5-pro" // default model
	}

	return &GeminiProvider{
		client:         client,
		model:          model,
		templateConfig: templateConfig,
	}, nil
}

// Name returns the provider name
func (g *GeminiProvider) Name() string {
	return "gemini"
}

// Model returns the model name being used
func (g *GeminiProvider) Model() string {
	return g.model
}

// Close closes the Gemini client
func (g *GeminiProvider) Close() error {
	return g.client.Close()
}

// ApplySuggestion uses Gemini to generate an adapted patch for the suggestion
func (g *GeminiProvider) ApplySuggestion(ctx context.Context, req *SuggestionRequest) (*SuggestionResponse, error) {
	// Build the prompt from template
	prompt, err := BuildPrompt(req, g.templateConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Call Gemini API
	model := g.client.GenerativeModel(g.model)

	// Configure model for JSON output
	model.ResponseMIMEType = "application/json"

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
	}

	// Parse the response
	return parseGeminiResponse(resp)
}

// parseGeminiResponse extracts the structured response from Gemini
func parseGeminiResponse(resp *genai.GenerateContentResponse) (*SuggestionResponse, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Extract text from the response
	var responseText string
	for _, part := range candidate.Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText += string(txt)
		}
	}

	if responseText == "" {
		return nil, fmt.Errorf("no text in Gemini response")
	}

	// Parse JSON response
	var result struct {
		Patch       string   `json:"patch"`
		Explanation string   `json:"explanation"`
		Confidence  float64  `json:"confidence"`
		Warnings    []string `json:"warnings"`
	}

	// Clean up response text (remove markdown code blocks if present)
	responseText = strings.TrimSpace(responseText)
	re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*)```")
	if matches := re.FindStringSubmatch(responseText); len(matches) > 1 {
		responseText = matches[1]
	}
	responseText = strings.TrimSpace(responseText)

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini JSON response: %w\nResponse: %s", err, responseText)
	}

	// Validate the response
	if result.Patch == "" {
		return nil, fmt.Errorf("gemini returned empty patch")
	}

	// Ensure warnings is not nil
	if result.Warnings == nil {
		result.Warnings = []string{}
	}

	return &SuggestionResponse{
		Patch:       result.Patch,
		Explanation: result.Explanation,
		Confidence:  result.Confidence,
		Warnings:    result.Warnings,
	}, nil
}
