# outrageous

This originally started as a port of the
[swarm](https://github.com/openai/swarm) agent sdk from Python to Go. The
functionality of an agent is still supported, but other features have and will
be added over time.

Supports:

- Clients support for models across Ollama, OpenAI, and Gemini (any OpenAI
  compatible API can be used)
- Compose agents across different models -- i.e. Sensitive information can be
  used only on local models
- Set of reusable tools that agents can use -- i.e Web Scraper
- Tool support for Struct fields and tags
- Working example ported from the original swarm -- i.e
  [function calling](examples/basic/function_calling/main.go)
- Experimental: Local vector store for embedding vectors

## Testing

All examples are tested against [Ollama](https://ollama.com) and
[llama3.1:8b](https://ollama.com/library/llama3.1).
