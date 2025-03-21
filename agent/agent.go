package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jtarchie/outrageous/client"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Agent struct {
	name         string
	instructions string
	Tools        Tools
	client       *client.Client
}

// AsTool creates a tool that returns the agent itself
// This is useful for agent handoff scenarios where one agent can call another agent
func (agent *Agent) AsTool(description string) Tool {
	return Tool{
		Name: toName(agent.name),
		Description: fmt.Sprintf(strings.TrimSpace(`
			You are a helpful agent that has the following instructions:

			%s
		`), description),
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
	return agent.name == "" && agent.instructions == "" && len(agent.Tools) == 0
}

// Identify an agent in the messages
func (agent *Agent) String() string {
	return fmt.Sprintf(`{"assistant": {"name": %q, "instructions": %q}}`, agent.name, agent.instructions)
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
		"agent_name", activeAgent.name,
		"max_messages", maxMessages,
		"initial_messages_count", len(messages),
		"tools_count", len(activeAgent.Tools),
	)

	for range maxMessages {
		if activeAgent.IsZero() {
			slog.Debug("agent.stopping",
				"reason", "agent_is_zero",
				"agent_name", activeAgent.name)
			break
		}

		completion := openai.ChatCompletionRequest{
			Model:    activeAgent.client.ModelName(),
			Messages: messages,
			Tools:    activeAgent.Tools.AsTools(),
		}

		slog.Debug("agent.requesting",
			"agent_name", activeAgent.name,
			"model", activeAgent.client.ModelName(),
			"tools_count", len(activeAgent.Tools),
		)

		response, err := activeAgent.client.CreateChatCompletion(
			ctx,
			completion,
		)
		if err != nil {
			return nil, fmt.Errorf("could not chat completion: %w", err)
		}

		message := response.Choices[0].Message
		message.Name = activeAgent.name

		// need to add toolcalls for gemini incompatibility
		// https://discuss.ai.google.dev/t/tool-calling-with-openai-api-not-working/60140/4
		for index := range message.ToolCalls {
			if message.ToolCalls[index].ID == "" {
				message.ToolCalls[index].ID = fmt.Sprintf("tool-%d", index)
			}
		}
		messages = append(messages, message)

		slog.Debug("agent.received",
			"agent_name", activeAgent.name,
			"has_content", message.Content != "",
			"tool_calls_count", len(message.ToolCalls))

		if len(message.ToolCalls) == 0 {
			slog.Debug("agent.completed",
				"agent_name", activeAgent.name,
				"reason", "no_tool_calls")
			break
		}

		for _, toolCall := range message.ToolCalls {
			functionName := toolCall.Function.Name

			slog.Debug("agent.tool_call",
				"agent_name", activeAgent.name,
				"tool_name", functionName,
				"tool_call_id", toolCall.ID)

			if tool, found := activeAgent.Tools.Get(functionName); found {
				params := map[string]any{}

				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal function arguments: %w", err)
				}

				slog.Debug("agent.executing_function",
					"agent_name", activeAgent.name,
					"tool_name", functionName)

				value, err := tool.Func(ctx, params)
				if err != nil {
					return nil, fmt.Errorf("could not call function: %w", err)
				}

				if nextAgent, ok := value.(*Agent); ok {
					slog.Debug("agent.handoff",
						"from_agent", activeAgent.name,
						"to_agent", nextAgent.name,
						"tool_name", functionName)
					if !nextAgent.IsZero() {
						activeAgent = nextAgent
					}
				} else {
					slog.Debug("agent.function_result",
						"agent_name", activeAgent.name,
						"tool_name", functionName,
						"result_type", fmt.Sprintf("%T", value))
				}

				messages = append(messages, Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Name:       functionName,
					Content:    fmt.Sprintf("%s", value),
				})

				break
			} else {
				slog.Debug("agent.missing_function",
					"agent_name", activeAgent.name,
					"tool_name", functionName)
				return nil, fmt.Errorf("tool not found: %s", toolCall.Function.Name)
			}
		}
	}

	slog.Debug("agent.run_completed",
		"agent_name", activeAgent.name,
		"final_messages_count", len(messages))

	return &Response{
		Messages: messages,
		Agent:    activeAgent,
	}, nil
}

type AgentOption func(*Agent)

func WithClient(client *client.Client) AgentOption {
	return func(agent *Agent) {
		agent.client = client
	}
}

func New(name, instructions string, options ...AgentOption) *Agent {
	agent := &Agent{
		name:         toName(name),
		instructions: instructions,
		client:       client.DefaultClient,
	}

	for _, option := range options {
		option(agent)
	}

	return agent
}
