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
	hooks        AgentHooks
	instructions string
	logger       *slog.Logger
	name         string
	schema       json.Marshaler

	Tools    Tools
	Handoffs Tools
}

func (agent *Agent) Name() string {
	return agent.name
}

func (agent *Agent) SetSchema(value any) error {
	var err error

	schema, ok := value.(json.Marshaler)
	if !ok {
		schema, err = jsonschema.GenerateSchemaForType(value)
		if err != nil {
			return fmt.Errorf("could not generate schema: %w", err)
		}
	}

	contents, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal schema: %w", err)
	}

	fmt.Println(string(contents))

	agent.schema = schema

	return nil
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

func ImageMessage(filename string) Message {
	imageBase64Encoded, _ := Base64EncodeImage(filename)

	return Message{
		Role: "user",
		MultiContent: []openai.ChatMessagePart{
			{
				Type: "image_url",
				ImageURL: &openai.ChatMessageImageURL{
					URL:    imageBase64Encoded,
					Detail: openai.ImageURLDetailAuto,
				},
			},
		},
	}
}

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

	// Call OnAgentStart hook if it exists
	if activeAgent.hooks != nil {
		if err := activeAgent.hooks.OnAgentStart(ctx, activeAgent, activeMessages); err != nil {
			return nil, fmt.Errorf("agent hook OnAgentStart failed: %w", err)
		}
	}

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

	response := &Response{
		Messages: messages,
		Agent:    activeAgent,
	}

	// Call OnAgentEnd hook if it exists
	if activeAgent.hooks != nil {
		if err := activeAgent.hooks.OnAgentEnd(ctx, activeAgent, response); err != nil {
			return nil, fmt.Errorf("agent hook OnAgentEnd failed: %w", err)
		}
	}

	return response, nil
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

	// Call OnToolStart hook if it exists
	if agent.hooks != nil {
		if err := agent.hooks.OnToolStart(ctx, agent, functionName, params); err != nil {
			return Message{}, nil, fmt.Errorf("agent hook OnToolStart failed: %w", err)
		}
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

	// Call OnToolEnd hook if it exists
	if agent.hooks != nil {
		if err := agent.hooks.OnToolEnd(ctx, agent, functionName, value); err != nil {
			return Message{}, nil, fmt.Errorf("agent hook OnToolEnd failed: %w", err)
		}
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
	var responseFormat *openai.ChatCompletionResponseFormat
	if agent.schema != nil {
		agent.logger.Debug("agent.schema", "agent_name", agent.name)
		responseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "schema",
				Schema: agent.schema,
				// Strict: true,
			},
		}
	}

	completeTools := append(agent.Tools.AsTools(), agent.Handoffs.AsTools()...)
	var toolChoice any = nil
	if len(completeTools) > 0 {
		toolChoice = "auto"
	}
	completion := openai.ChatCompletionRequest{
		Model:          agent.client.ModelName(),
		Messages:       activeMessages,
		Tools:          completeTools,
		ToolChoice:     toolChoice,
		ResponseFormat: responseFormat,
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
	agentContext, _ := params["agent_context"].(string)

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

	// Create temporary response for the current agent
	currentAgentResponse := &Response{
		Messages: messages,
		Agent:    agent,
	}

	// Call OnAgentEnd hook for the current agent
	if agent.hooks != nil {
		if err := agent.hooks.OnAgentEnd(context.Background(), agent, currentAgentResponse); err != nil {
			agent.logger.Error("agent.handoff_end_hook_failed", "error", err)
		}
	}

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

	// Call OnHandoff hook if it exists
	if agent.hooks != nil {
		if err := agent.hooks.OnHandoff(context.Background(), agent, nextAgent, agentContext); err != nil {
			agent.logger.Error("agent.handoff_hook_failed", "error", err)
		}
	}

	// Call OnAgentStart hook for the next agent
	if nextAgent.hooks != nil {
		if err := nextAgent.hooks.OnAgentStart(context.Background(), nextAgent, activeMessages); err != nil {
			agent.logger.Error("agent.handoff_start_hook_failed", "error", err)
		}
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

// WithHooks adds lifecycle hooks to the agent
func WithHooks(hooks AgentHooks) AgentOption {
	return func(agent *Agent) {
		agent.hooks = hooks
	}
}

func New(name, instructions string, options ...AgentOption) *Agent {
	agent := &Agent{
		name:         toFormattedName(name),
		instructions: instructions,
		client:       client.DefaultClient,
		logger:       slog.Default().WithGroup("agent"),
		hooks:        &DefaultAgentHooks{}, // Default no-op hooks
	}

	for _, option := range options {
		option(agent)
	}

	return agent
}
