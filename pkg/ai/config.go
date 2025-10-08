package ai

import (
	"fmt"
	"os"
)

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
	config := &Config{
		Provider: getEnvWithDefault("GH_REVIEW_AI_PROVIDER", "gemini"),
		Model:    os.Getenv("GH_REVIEW_AI_MODEL"),
	}

	// Load API key based on provider
	switch config.Provider {
	case "gemini":
		config.APIKey = os.Getenv("GEMINI_API_KEY")
		if config.APIKey == "" {
			config.APIKey = os.Getenv("GOOGLE_API_KEY") // Alternative
		}
	case "openai":
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	case "claude":
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Load custom template path if set
	config.CustomTemplatePath = os.Getenv("GH_REVIEW_AI_TEMPLATE")

	return config
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
