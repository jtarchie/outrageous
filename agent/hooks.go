package agent

import "context"

// AgentHooks defines callbacks for agent lifecycle events
type AgentHooks interface {
	OnAgentStart(ctx context.Context, agent *Agent, messages Messages) error
	OnAgentEnd(ctx context.Context, agent *Agent, response *Response) error
	OnHandoff(ctx context.Context, fromAgent *Agent, toAgent *Agent, agentContext string) error
	OnToolStart(ctx context.Context, agent *Agent, toolName string, params map[string]any) error
	OnToolEnd(ctx context.Context, agent *Agent, toolName string, result any) error
}

// DefaultAgentHooks provides a no-op implementation of AgentHooks
// Users can embed this struct and override only the methods they need
type DefaultAgentHooks struct{}

func (h *DefaultAgentHooks) OnAgentStart(ctx context.Context, agent *Agent, messages Messages) error {
	return nil
}

func (h *DefaultAgentHooks) OnAgentEnd(ctx context.Context, agent *Agent, response *Response) error {
	return nil
}

func (h *DefaultAgentHooks) OnHandoff(ctx context.Context, fromAgent *Agent, toAgent *Agent, agentContext string) error {
	return nil
}

func (h *DefaultAgentHooks) OnToolStart(ctx context.Context, agent *Agent, toolName string, params map[string]any) error {
	return nil
}

func (h *DefaultAgentHooks) OnToolEnd(ctx context.Context, agent *Agent, toolName string, result any) error {
	return nil
}
