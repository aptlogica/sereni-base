package main

import (
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
)

// initConfig sets up Viper to read configuration from a .env file and
// from the real environment. The .env file is loaded first so local
// development uses the checked-in configuration by default.
func initConfig() error {
	viper.SetConfigFile(".env") // .env file in the project root
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		// If the file doesn't exist, we still allow using system env vars.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading .env file: %w", err)
		}
	}

	return nil
}

// loadSystemPrompt reads the system prompt from the given file path.
func loadSystemPrompt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading %s file: %w", path, err)
	}
	return string(data), nil
}

// newOpenAIClientFromEnv creates an OpenAI client using the OPENAI_API_KEY
// read from .env first, then from the process environment as a fallback.
func newOpenAIClientFromEnv() (*openai.Client, error) {
	apiKey := viper.GetString("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))
	return client, nil
}
