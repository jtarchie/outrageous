package client

import openai "github.com/sashabaranov/go-openai"

// https://docs.anthropic.com/en/api/openai-sdk
// models: https://docs.anthropic.com/en/docs/about-claude/models/all-models
func NewAnthropicClient(apiToken, model string) *Client {
	config := openai.DefaultConfig(apiToken)
	config.BaseURL = "https://api.anthropic.com/v1/"

	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}
