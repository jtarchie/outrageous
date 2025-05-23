package client

// https://github.com/ollama/ollama/blob/main/docs/openai.md
// models: https://ollama.com/search
func NewOllamaClient(model string) *Client {
	return New("http://localhost:11434/v1", "", model)
}
