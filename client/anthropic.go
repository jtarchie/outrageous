package client

// https://docs.anthropic.com/en/api/openai-sdk
// models: https://docs.anthropic.com/en/docs/about-claude/models/all-models
func NewAnthropicClient(apiToken, model string) *Client {
	return New("https://api.anthropic.com/v1/", apiToken, model)
}
