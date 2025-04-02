package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jtarchie/outrageous/client"
	"github.com/samber/lo"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Agent struct {
	client       *client.Client
	instructions string
	logger       *slog.Logger
	name         string

	Tools    Tools
	Handoffs Tools
}

func (agent *Agent) Name() string {
	return agent.name
}

// AsTool creates a tool that returns the agent itself
// This is useful for agent handoff scenarios where one agent can call another agent
func (agent *Agent) AsTool(description string) Tool {
	return Tool{
		Name: toFormattedName(agent.name),
		Description: fmt.Sprintf(strings.TrimSpace(`
			Agent Handoff Tool: Use this to transfer the conversation to the %s agent.
			
			This agent has the following instructions:
			%s
			
			IMPORTANT: When calling this tool, you must provide detailed context about the 
			current conversation state and why you're handing off to this agent. The receiving 
			agent will use this context to continue the conversation seamlessly.
		`), agent.name, description),
		Parameters: &jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"agent_context": {
					Type:        jsonschema.String,
					Description: "Required detailed context about the current conversation state, including what has been discussed, any relevant information gathered, and why you're handing off to this agent. This will help the receiving agent continue the conversation effectively.",
				},
			},
			Required: []string{"agent_context"},
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
	logger := agent.logger

	maxMessages := 10
	activeAgent := agent

	messages = append(
		Messages{
			Message{
				Role:    "system",
				Content: activeAgent.instructions,
			},
		},
		messages...,
	)

	activeMessages := make(Messages, len(messages))
	_ = copy(activeMessages, messages)

	logger.Debug("agent.starting",
		"agent_name", activeAgent.name,
		"max_messages", maxMessages,
		"initial_messages_count", len(messages),
		"tools_count", len(activeAgent.Tools),
		"handoffs_count", len(activeAgent.Handoffs),
		"active_messages_count", len(activeMessages),
	)

	for range maxMessages {
		if activeAgent.IsZero() {
			logger.Debug("agent.stopping",
				"reason", "agent_is_zero",
				"agent_name", activeAgent.name)
			break
		}

		completeTools := append(activeAgent.Tools.AsTools(), activeAgent.Handoffs.AsTools()...)
		completion := openai.ChatCompletionRequest{
			Model:      activeAgent.client.ModelName(),
			Messages:   activeMessages,
			Tools:      completeTools,
			ToolChoice: "auto",
		}

		logger.Debug("agent.requesting",
			"agent_name", activeAgent.name,
			"model", activeAgent.client.ModelName(),
			"tools_count", len(activeAgent.Tools),
			"handoffs_count", len(activeAgent.Handoffs),
			"messages_count", len(activeMessages),
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
		activeMessages = append(activeMessages, message)

		logger.Debug("agent.received",
			"agent_name", activeAgent.name,
			"has_content", message.Content != "",
			"tool_calls_count", len(message.ToolCalls))

		if len(message.ToolCalls) == 0 {
			logger.Debug("agent.completed",
				"agent_name", activeAgent.name,
				"reason", "no_tool_calls")
			break
		}

		for _, toolCall := range message.ToolCalls {
			functionName := toolCall.Function.Name

			logger.Debug("agent.tool_call",
				"agent_name", activeAgent.name,
				"tool_name", functionName,
				"tool_call_id", toolCall.ID)

			if tool, found := activeAgent.Tools.Get(functionName); found {
				params := map[string]any{}

				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal function arguments: %w", err)
				}

				logger.Debug("agent.executing_function",
					"agent_name", activeAgent.name,
					"tool_name", functionName)

				value, err := tool.Func(ctx, params)
				if err != nil {
					return nil, fmt.Errorf("could not call function: %w", err)
				}

				logger.Debug("agent.function_result",
					"agent_name", activeAgent.name,
					"tool_name", functionName,
					"result_type", fmt.Sprintf("%T", value))

				message := Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Name:       functionName,
					Content:    fmt.Sprintf("%s", value),
				}
				messages = append(messages, message)
				activeMessages = append(activeMessages, message)

				break
			} else if agent, found := activeAgent.Handoffs.Get(functionName); found {
				params := map[string]any{}

				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal function arguments: %w", err)
				}

				logger.Debug("agent.handoff_agent",
					"agent_name", activeAgent.name,
					"tool_name", functionName)

				value, err := agent.Func(ctx, params)
				if err != nil {
					return nil, fmt.Errorf("could not call function: %w", err)
				}

				logger.Debug("agent.handoff_result",
					"agent_name", activeAgent.name,
					"tool_name", functionName,
					"result_type", fmt.Sprintf("%T", value))

				message := Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Name:       functionName,
					Content:    fmt.Sprintf("%s", value),
				}
				messages = append(messages, message)
				activeMessages = append(activeMessages, message)

				if nextAgent, ok := value.(*Agent); ok {
					logger.Debug("agent.handoff",
						"from_agent", activeAgent.name,
						"to_agent", nextAgent.name,
						"tool_name", functionName)
					if !nextAgent.IsZero() {
						lastUserMessage, _ := lo.Find(messages, func(m Message) bool {
							return m.Role == "user"
						})
						agentContext := params["agent_context"]

						logger.Debug("agent.handoff_context",
							"from_agent", activeAgent.name,
							"to_agent", nextAgent.name,
							"last_user_message", lastUserMessage.Content,
							"agent_context", agentContext,
						)

						activeMessages = Messages{
							Message{
								Role: "system",
								Content: fmt.Sprintf(strings.TrimSpace(`%s
									This is a conversation starting from another agent. The conversation have been passed to you.
									The previous agent has provide context for you to follow: %s
								`), nextAgent.instructions, agentContext),
							},
							lastUserMessage,
						}
						activeAgent = nextAgent
					}
				}

				break
			} else {
				logger.Debug("agent.missing_function",
					"agent_name", activeAgent.name,
					"tool_name", functionName)
				return nil, fmt.Errorf("tool not found: %s", toolCall.Function.Name)
			}
		}
	}

	logger.Debug("agent.run_completed",
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

func WithLogger(logger *slog.Logger) AgentOption {
	return func(agent *Agent) {
		agent.logger = logger.WithGroup("agent")
	}
}

func New(name, instructions string, options ...AgentOption) *Agent {
	agent := &Agent{
		name:         toFormattedName(name),
		instructions: instructions,
		client:       client.DefaultClient,
		logger:       slog.Default().WithGroup("agent"),
	}

	for _, option := range options {
		option(agent)
	}

	return agent
}
