package client

import openai "github.com/sashabaranov/go-openai"

func NewOpenAIClient(apiToken, model string) *Client {
	config := openai.DefaultConfig(apiToken)
	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}
