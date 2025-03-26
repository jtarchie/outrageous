package client

import (
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	*openai.Client
	model string
}

func (c *Client) ModelName() string {
	return c.model
}

// this is a local model that can do function calling
var DefaultClient *Client = NewOllamaClient("llama3.1")
