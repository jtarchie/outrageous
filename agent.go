package outrageous

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Agent struct {
	Name         string
	Instructions string
	Functions    Functions
	Client       *Client
}

// AsFunction creates a function that returns the agent itself
// This is useful for agent handoff scenarios where one agent can call another agent
func (agent *Agent) AsFunction(description string) Function {
	return Function{
		Name:        agent.Name,
		Description: description,
		Parameters: &jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"instruction": {
					Type:        jsonschema.String,
					Description: "Instruction for the agent from the previous agent",
				},
			},
			Required: []string{},
		},
		Func: func(ctx context.Context, params map[string]any) (any, error) {
			return agent, nil
		},
	}
}

// Know when the agent has no responsibility
func (agent *Agent) IsZero() bool {
	return agent.Name == "" && agent.Instructions == "" && len(agent.Functions) == 0
}

// Identify an agent in the messages
func (agent *Agent) String() string {
	return fmt.Sprintf(`{"assistant": {"name": %q, "instructions": %q}}`, agent.Name, agent.Instructions)
}

type Message = openai.ChatCompletionMessage
type Messages []Message

type Response struct {
	Messages Messages
	Agent    *Agent
}

// Run executes the agent's logic
// It will continue to run until the agent has no more messages to process
// or the maximum number of messages is reached.
func (agent *Agent) Run(ctx context.Context, messages Messages) (*Response, error) {
	maxMessages := 10
	activeAgent := agent

	slog.Debug("agent.starting",
		"agent_name", activeAgent.Name,
		"max_messages", maxMessages,
		"initial_messages_count", len(messages),
		"functions_count", len(activeAgent.Functions),
	)

	for range maxMessages {
		if activeAgent.IsZero() {
			slog.Debug("agent.stopping",
				"reason", "agent_is_zero",
				"agent_name", activeAgent.Name)
			break
		}

		completion := openai.ChatCompletionRequest{
			Model:    activeAgent.Client.model,
			Messages: messages,
			Tools:    activeAgent.Functions.AsTools(),
		}

		slog.Debug("agent.requesting",
			"agent_name", activeAgent.Name,
			"model", activeAgent.Client.model,
			"functions_count", len(activeAgent.Functions),
		)

		response, err := activeAgent.Client.CreateChatCompletion(
			ctx,
			completion,
		)
		if err != nil {
			return nil, fmt.Errorf("could not chat completion: %w", err)
		}

		message := response.Choices[0].Message
		message.Name = activeAgent.Name
		messages = append(messages, message)

		slog.Debug("agent.received",
			"agent_name", activeAgent.Name,
			"has_content", message.Content != "",
			"tool_calls_count", len(message.ToolCalls))

		if len(message.ToolCalls) == 0 {
			slog.Debug("agent.completed",
				"agent_name", activeAgent.Name,
				"reason", "no_tool_calls")
			break
		}

		for _, toolCall := range message.ToolCalls {
			functionName := toolCall.Function.Name

			slog.Debug("agent.tool_call",
				"agent_name", activeAgent.Name,
				"function_name", functionName,
				"tool_call_id", toolCall.ID)

			if function, found := activeAgent.Functions.Get(functionName); found {
				params := map[string]any{}

				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal function arguments: %w", err)
				}

				slog.Debug("agent.executing_function",
					"agent_name", activeAgent.Name,
					"function_name", functionName)

				value, err := function.Func(ctx, params)
				if err != nil {
					return nil, fmt.Errorf("could not call function: %w", err)
				}

				if nextAgent, ok := value.(*Agent); ok {
					slog.Debug("agent.handoff",
						"from_agent", activeAgent.Name,
						"to_agent", nextAgent.Name,
						"function_name", functionName)
					if !nextAgent.IsZero() {
						activeAgent = nextAgent
					}
				} else {
					slog.Debug("agent.function_result",
						"agent_name", activeAgent.Name,
						"function_name", functionName,
						"result_type", fmt.Sprintf("%T", value))
				}

				messages = append(messages, Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
					Content:    fmt.Sprintf("%s", value),
				})

				break
			} else {
				slog.Debug("agent.missing_function",
					"agent_name", activeAgent.Name,
					"function_name", functionName)
				return nil, fmt.Errorf("function not found: %s", toolCall.Function.Name)
			}
		}
	}

	slog.Debug("agent.run_completed",
		"agent_name", activeAgent.Name,
		"final_messages_count", len(messages))

	return &Response{
		Messages: messages,
		Agent:    activeAgent,
	}, nil
}

func NewAgent(name, instructions string) *Agent {
	return &Agent{
		Name:         name,
		Instructions: instructions,
		Client:       DefaultClient,
	}
}
