package client

// htthttps://openrouter.ai/docs/quickstart
func NewOpenRouterClient(apiToken, model string) *Client {
	return New("https://openrouter.ai/api/v1", apiToken, model)
}
