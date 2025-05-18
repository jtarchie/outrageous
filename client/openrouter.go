package client

import openai "github.com/sashabaranov/go-openai"

// htthttps://openrouter.ai/docs/quickstart
func NewOpenRouterClient(apiToken, model string) *Client {
	config := openai.DefaultConfig(apiToken)
	config.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}
