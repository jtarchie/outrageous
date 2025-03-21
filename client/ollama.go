package client

import openai "github.com/sashabaranov/go-openai"

func NewOllamaClient(model string) *Client {
	config := openai.DefaultConfig("")
	config.BaseURL = "http://localhost:11434/v1"

	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}
