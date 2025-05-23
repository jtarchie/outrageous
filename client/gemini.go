package client

// https://ai.google.dev/gemini-api/docs/openai
// models: https://ai.google.dev/gemini-api/docs/models
func NewGeminiClient(apiToken, model string) *Client {
	return New("https://generativelanguage.googleapis.com/v1beta/openai", apiToken, model)
}
