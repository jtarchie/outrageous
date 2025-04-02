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

	// Initialize messages with system instruction
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
	copy(activeMessages, messages)

	logger.Debug("agent.starting",
		"agent_name", activeAgent.name,
		"max_messages", maxMessages,
		"initial_messages_count", len(messages),
		"tools_count", len(activeAgent.Tools),
		"handoffs_count", len(activeAgent.Handoffs),
		"active_messages_count", len(activeMessages),
	)

	for range maxMessages {
		// Exit if we have no active agent
		if activeAgent.IsZero() {
			logger.Debug("agent.stopping", "reason", "agent_is_zero", "agent_name", activeAgent.name)
			break
		}

		// Get AI response
		message, err := activeAgent.getModelResponse(ctx, activeMessages)
		if err != nil {
			return nil, err
		}

		// Add response to message history
		messages = append(messages, message)
		activeMessages = append(activeMessages, message)

		// If no tool calls, we're done
		if len(message.ToolCalls) == 0 {
			logger.Debug("agent.completed", "agent_name", activeAgent.name, "reason", "no_tool_calls")
			break
		}

		// Process the first tool call
		toolCall := message.ToolCalls[0]
		responseMsg, nextAgent, err := activeAgent.processToolCall(ctx, toolCall)
		if err != nil {
			return nil, err
		}

		// Add tool response to messages
		messages = append(messages, responseMsg)
		activeMessages = append(activeMessages, responseMsg)

		// Handle agent handoff if needed
		if nextAgent != nil {
			activeAgent, activeMessages = activeAgent.prepareHandoff(nextAgent, messages, toolCall.Function.Arguments)
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

// processToolCall handles both regular tools and handoff tools with unified logic
func (agent *Agent) processToolCall(ctx context.Context, toolCall openai.ToolCall) (Message, *Agent, error) {
	functionName := toolCall.Function.Name
	agent.logger.Debug("agent.tool_call",
		"agent_name", agent.name,
		"tool_name", functionName,
		"tool_call_id", toolCall.ID)

	// Parse parameters once
	params := map[string]any{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return Message{}, nil, fmt.Errorf("could not unmarshal function arguments: %w", err)
	}

	var value any
	var err error
	var nextAgent *Agent
	resultType := "function_result"

	// Check if it's a regular tool or handoff
	if tool, found := agent.Tools.Get(functionName); found {
		agent.logger.Debug("agent.executing_function",
			"agent_name", agent.name,
			"tool_name", functionName)
		value, err = tool.Func(ctx, params)
	} else if handoffTool, found := agent.Handoffs.Get(functionName); found {
		resultType = "handoff_result"
		agent.logger.Debug("agent.handoff_agent",
			"agent_name", agent.name,
			"tool_name", functionName)
		value, err = handoffTool.Func(ctx, params)

		// Check if we need to switch agents
		if agentValue, ok := value.(*Agent); ok && !agentValue.IsZero() {
			nextAgent = agentValue
			agent.logger.Debug("agent.handoff",
				"from_agent", agent.name,
				"to_agent", nextAgent.name,
				"tool_name", functionName)
		}
	} else {
		agent.logger.Debug("agent.missing_function",
			"agent_name", agent.name,
			"tool_name", functionName)
		return Message{}, nil, fmt.Errorf("tool not found: %s", functionName)
	}

	if err != nil {
		return Message{}, nil, fmt.Errorf("could not call function: %w", err)
	}

	agent.logger.Debug(fmt.Sprintf("agent.%s", resultType),
		"agent_name", agent.name,
		"tool_name", functionName,
		"result_type", fmt.Sprintf("%T", value))

	// Create the response message
	message := Message{
		Role:       "tool",
		ToolCallID: toolCall.ID,
		Name:       functionName,
		Content:    fmt.Sprintf("%s", value),
	}

	return message, nextAgent, nil
}

// getModelResponse gets a response from the LLM
func (agent *Agent) getModelResponse(ctx context.Context, activeMessages Messages) (Message, error) {
	completeTools := append(agent.Tools.AsTools(), agent.Handoffs.AsTools()...)
	completion := openai.ChatCompletionRequest{
		Model:      agent.client.ModelName(),
		Messages:   activeMessages,
		Tools:      completeTools,
		ToolChoice: "auto",
	}

	agent.logger.Debug("agent.requesting",
		"agent_name", agent.name,
		"model", agent.client.ModelName(),
		"tools_count", len(agent.Tools),
		"handoffs_count", len(agent.Handoffs),
		"messages_count", len(activeMessages),
	)

	response, err := agent.client.CreateChatCompletion(ctx, completion)
	if err != nil {
		return Message{}, fmt.Errorf("could not chat completion: %w", err)
	}

	message := response.Choices[0].Message
	message.Name = agent.name

	// Fix for Gemini compatibility
	for index := range message.ToolCalls {
		if message.ToolCalls[index].ID == "" {
			message.ToolCalls[index].ID = fmt.Sprintf("tool-%d", index)
		}
	}

	agent.logger.Debug("agent.received",
		"agent_name", agent.name,
		"has_content", message.Content != "",
		"tool_calls_count", len(message.ToolCalls))

	return message, nil
}

// prepareHandoff prepares the handoff to another agent
func (agent *Agent) prepareHandoff(nextAgent *Agent, messages Messages, argJSON string) (*Agent, Messages) {
	// Parse the arguments to get the agent context
	params := map[string]any{}
	_ = json.Unmarshal([]byte(argJSON), &params)
	agentContext := params["agent_context"]

	// Find the last user message
	lastUserMessage, _ := lo.Find(messages, func(m Message) bool {
		return m.Role == "user"
	})

	agent.logger.Debug("agent.handoff_context",
		"from_agent", agent.name,
		"to_agent", nextAgent.name,
		"last_user_message", lastUserMessage.Content,
		"agent_context", agentContext,
	)

	// Create new messages for the next agent
	activeMessages := Messages{
		Message{
			Role: "system",
			Content: fmt.Sprintf(strings.TrimSpace(`%s
                This is a conversation starting from another agent. The conversation have been passed to you.
                The previous agent has provide context for you to follow: %s
            `), nextAgent.instructions, agentContext),
		},
		lastUserMessage,
	}

	return nextAgent, activeMessages
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
