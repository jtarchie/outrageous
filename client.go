package outrageous

import (
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	*openai.Client
	model string
}

func ollamaConfig() openai.ClientConfig {
	config := openai.DefaultConfig("")
	config.BaseURL = "http://localhost:11434/v1"
	return config
}

func NewOllamaClient(model string) *Client {
	client := openai.NewClientWithConfig(ollamaConfig())

	return &Client{
		Client: client,
		model:  model,
	}
}

var DefaultClient *Client = NewOllamaClient("llama3.2")
