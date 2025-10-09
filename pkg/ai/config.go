package ai

import (
	"fmt"
	"os"
)

// ProviderMetadata holds information about an AI provider.
type ProviderMetadata struct {
	Label   string
	EnvVars []string
}

// providerInfo maps provider names to their metadata.
var providerInfo = map[string]ProviderMetadata{
	"gemini": {"Gemini", []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}},
	"openai": {"OpenAI", []string{"OPENAI_API_KEY"}},    // Planned for future support
	"claude": {"Claude", []string{"ANTHROPIC_API_KEY"}}, // Planned for future support
}

// GetProviderMetadata returns metadata for a given provider.
func GetProviderMetadata(provider string) (ProviderMetadata, bool) {
	info, ok := providerInfo[provider]
	return info, ok
}

// Config holds AI provider configuration
type Config struct {
	Provider           string
	Model              string
	APIKey             string
	CustomTemplatePath string
	CustomVariables    map[string]interface{}
}

// NewProviderFromConfig creates an AI provider based on configuration
func NewProviderFromConfig(config *Config) (AIProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	templateConfig := &TemplateConfig{
		CustomTemplatePath: config.CustomTemplatePath,
		CustomVariables:    config.CustomVariables,
	}

	switch config.Provider {
	case "gemini":
		return NewGeminiProvider(config.APIKey, config.Model, templateConfig)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s (supported: gemini)", config.Provider)
	}
}

// LoadConfigFromEnv loads AI configuration from environment variables
func LoadConfigFromEnv() *Config {
	provider := getEnvWithDefault("GH_PRREVIEW_AI_PROVIDER", "")

	config := &Config{
		Provider: provider,
		Model:    os.Getenv("GH_PRREVIEW_AI_MODEL"),
	}

	// Load API key based on provider
	if meta, ok := GetProviderMetadata(config.Provider); ok {
		for _, envVar := range meta.EnvVars {
			if key := os.Getenv(envVar); key != "" {
				config.APIKey = key
				break
			}
		}
	}

	// Load custom template path if set
	config.CustomTemplatePath = os.Getenv("GH_PRREVIEW_AI_TEMPLATE")
	return config
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
