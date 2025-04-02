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

## Installation

```bash
go get github.com/jtarchie/outrageous
```

## Examples

### Building an Agent

```go
package main

import (
    "context"
    "log"
    "fmt"
    . "github.com/jtarchie/outrageous/agent"
)

func main() {
    // Create a new agent with a name and instructions
    // By default, it uses the Ollama client with llama3.1 model
    agent := New(
        "Helper Agent",
        "You are a helpful assistant that provides concise answers.",
    )

    // Run the agent with a user message
    // The agent will process the message and generate a response
    response, err := agent.Run(
        context.Background(),
        Messages{
            Message{Role: "user", Content: "What are the benefits of Go for AI applications?"},
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    // Print the last message from the agent
    fmt.Println(response.Messages[len(response.Messages)-1].Content)
}
```

### Creating a custom tool

```go
package main

import (
    "context"
    "fmt"
    "log"
    . "github.com/jtarchie/outrageous/agent"
)

// Define a struct that implements the Caller interface
// The JSON tags are used to map parameters from the LLM to struct fields
type Calculator struct {
    Operation string  `json:"operation"` // Required operation (add, subtract, multiply, divide)
    A         float64 `json:"a"`         // First number
    B         float64 `json:"b"`         // Second number
}

// Implement the Call method required by the Caller interface
func (c Calculator) Call(ctx context.Context) (any, error) {
    switch c.Operation {
    case "add":
        return fmt.Sprintf("%.2f", c.A+c.B), nil
    case "subtract":
        return fmt.Sprintf("%.2f", c.A-c.B), nil
    case "multiply":
        return fmt.Sprintf("%.2f", c.A*c.B), nil
    case "divide":
        if c.B == 0 {
            return nil, fmt.Errorf("division by zero")
        }
        return fmt.Sprintf("%.2f", c.A/c.B), nil
    default:
        return nil, fmt.Errorf("unknown operation: %s", c.Operation)
    }
}

func main() {
    agent := New(
        "Math Assistant",
        "You help users perform mathematical calculations.",
    )

    // Wrap the Calculator struct as a tool and add it to the agent
    // MustWrapStruct panics if there's an error, use WrapStruct for error handling
    agent.Tools.Add(
        MustWrapStruct(
            "Perform basic arithmetic operations like add, subtract, multiply, and divide",
            Calculator{},
        ),
    )

    response, err := agent.Run(
        context.Background(),
        Messages{
            Message{Role: "user", Content: "What is 42 multiplied by 0.5?"},
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Messages[len(response.Messages)-1].Content)
}
```

### Using a Different Client

```go
package main

import (
    "context"
    "log"
    "os"
    "fmt"

    . "github.com/jtarchie/outrageous/agent"
    "github.com/jtarchie/outrageous/client"
)

func main() {
    // Create a custom OpenAI client
    // Replace with your API key from environment variable
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable is required")
    }
    
    // Create a new agent with the custom client
    agent := New(
        "GPT-4 Agent",
        "You are an advanced assistant powered by GPT-4.",
        WithClient(client.NewOpenAIClient(apiKey, "gpt-4o")), // Override the default Ollama client
    )
    
    response, err := agent.Run(
        context.Background(),
        Messages{
            Message{Role: "user", Content: "Explain quantum computing in simple terms."},
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Messages[len(response.Messages)-1].Content)
}
```

These are examples of using different clients:

```go
package main

import (
    "os"
    "github.com/jtarchie/outrageous/client"
)

var openAIClient = client.NewGeminiClient(os.Getenv("OPENAI_API_KEY"), "gpt-4o")
var ollamaClient = client.NewOllamaClient("llama3.2") // this is our default client
var geminiClient = client.NewGeminiClient(os.Getenv("GEMINI_API_TOKEN"), "gemini-2.0-flash")
var claudeClient =  client.NewAnthropicClient(os.Getenv("ANTHROPIC_API_TOKEN"), "claude-3-5-sonnet-latest")
```

### Using Agent Lifecycle Hooks

Please see [example](examples/basic/hooks/main.go) for more information.

The current support hooks:

- `OnAgentStart`: Triggered when an agent begins execution, before any model
  calls are made. This occurs at the start of the `Run` method and after an
  agent handoff has been prepared.
- `OnToolStart`: Called immediately before a tool or handoff function is
  executed, after the parameters have been parsed from the LLM's function call.
- `OnToolEnd`: Invoked after a tool or handoff function completes successfully,
  providing access to the tool's return value before it's formatted as a
  message.
- `OnHandoff`: Called during an agent handoff process, after the current agent
  has finished but before the new agent fully starts processing. This happens
  after the preparation of context for the handoff.
- `OnAgentEnd`: Triggered when an agent completes its execution or when an agent
  is about to hand off to another agent. This happens at the end of the `Run`
  method or during the handoff preparation.

## Testing

All [examples](examples/) are tested against [Ollama](https://ollama.com) and
[llama3.2:3b](https://ollama.com/library/llama3.2).

```bash
brew bundle # should work on Linux and Mac

# in a tab far away
ollama start

#in another tab
ollama pull llama3.2
task test
```
