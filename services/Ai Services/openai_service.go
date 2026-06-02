package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
)

// extractSchema calls the OpenAI Chat Completions API with the provided system
// prompt and user input, and attempts to normalize the response into
// pretty-printed JSON describing the schema.
func extractSchema(client *openai.Client, systemPrompt, userInput string) (string, error) {
	// fmt.Println("systemPrompt----->:", systemPrompt)
	ctx := context.Background()

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userInput),
		}),
		Model: openai.F(openai.ChatModelGPT4o),
	})
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := chatCompletion.Choices[0].Message.Content

	// Try to parse and pretty print the JSON
	var schema SchemaResponse
	// fmt.Println("content----->:", content)
	// Extract JSON from markdown code blocks if present
	jsonStr := extractJSON(content)
	// fmt.Println("jsonStr----->:", string(jsonStr))
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		// If parsing fails, return the raw response
		return content, nil
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return content, nil
	}
	// fmt.Println("prettyJSON----->:", string(prettyJSON))
	return string(prettyJSON), nil
}

func extractBaseSchema(client *openai.Client, systemPrompt, userInput string) (string, error) {
	ctx := context.Background()

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userInput),
		}),
		Model: openai.F(openai.ChatModelGPT4o),
	})
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := chatCompletion.Choices[0].Message.Content

	jsonStr := extractJSON(content)
	
	var v any
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return content, nil
	}
	prettyJSON, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return content, nil
	}
	return string(prettyJSON), nil
}

// extractCSVOutput calls the OpenAI Chat Completions API for CSV-formatted output.
func extractCSVOutput(client *openai.Client, systemPrompt, userInput string) (string, error) {
	ctx := context.Background()

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userInput),
		}),
		Model: openai.F(openai.ChatModelGPT4o),
	})
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := strings.TrimSpace(chatCompletion.Choices[0].Message.Content)
	return content, nil
}

// extractJSON strips markdown code fences (if present) and returns the
// JSON portion of the content.
func extractJSON(content string) string {
	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)

	// Check for ```json or ``` blocks
	if strings.Contains(content, "```") {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start != -1 && end != -1 && end > start {
			return content[start : end+1]
		}
	}

	return content
}
