package client

import openai "github.com/sashabaranov/go-openai"

// https://ai.google.dev/gemini-api/docs/openai
// models: https://ai.google.dev/gemini-api/docs/models
func NewGeminiClient(apiToken, model string) *Client {
	config := openai.DefaultConfig(apiToken)
	config.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}
