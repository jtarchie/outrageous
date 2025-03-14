package outrageous

import (
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	*openai.Client
	model string
}

func NewOllamaClient(model string) *Client {
	config := openai.DefaultConfig("")
	config.BaseURL = "http://localhost:11434/v1"

	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}

// https://ai.google.dev/gemini-api/docs/openai
func NewGeminiClient(apiToken, model string) *Client {
	config := openai.DefaultConfig(apiToken)
	config.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}

func NewOpenAIClient(apiToken, model string) *Client {
	config := openai.DefaultConfig(apiToken)
	client := openai.NewClientWithConfig(config)

	return &Client{
		Client: client,
		model:  model,
	}
}

// this is a local model that can do function calling
var DefaultClient *Client = NewOllamaClient("llama3.2")
