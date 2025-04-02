package main

import (
	"context"
	"fmt"
	"github.com/jtarchie/outrageous/agent"
	"log"
	"time"
)

// Create custom hooks by embedding DefaultAgentHooks
// so you only need to implement the methods you care about
type DebugHooks struct {
	agent.DefaultAgentHooks
	startTime time.Time
}

// Track when the agent starts processing
func (h *DebugHooks) OnAgentStart(ctx context.Context, agent *agent.Agent, messages agent.Messages) error {
	h.startTime = time.Now()
	fmt.Printf("Agent '%s' started at %s\n", agent.Name(), h.startTime.Format(time.RFC3339))
	return nil
}

// Measure execution time when the agent completes
func (h *DebugHooks) OnAgentEnd(ctx context.Context, agent *agent.Agent, response *agent.Response) error {
	duration := time.Since(h.startTime)
	fmt.Printf("Agent '%s' completed in %s\n", agent.Name(), duration)
	return nil
}

// Log when tools are used
func (h *DebugHooks) OnToolStart(ctx context.Context, agent *agent.Agent, toolName string, params map[string]any) error {
	fmt.Printf("Agent '%s' using tool '%s' with params: %v\n", agent.Name(), toolName, params)
	return nil
}

// Log the results of tool executions
func (h *DebugHooks) OnToolEnd(ctx context.Context, agent *agent.Agent, toolName string, result any) error {
	fmt.Printf("Tool '%s' returned: %v\n", toolName, result)
	return nil
}

// Track agent handoffs
func (h *DebugHooks) OnHandoff(ctx context.Context, fromAgent *agent.Agent, toAgent *agent.Agent, agentContext string) error {
	fmt.Printf("Handoff from '%s' to '%s'\n", fromAgent.Name(), toAgent.Name())
	fmt.Printf("Handoff context: %s\n", agentContext)
	return nil
}

func main() {
	// Create an agent with custom hooks
	helperAgent := agent.New(
		"Helper Agent",
		"You are a helpful assistant that provides concise answers.",
		agent.WithHooks(&DebugHooks{}),
	)

	// Add a simple tool
	helperAgent.Tools.Add(agent.Tool{
		Name:        "get_current_time",
		Description: "Get the current server time",
		Func: func(ctx context.Context, params map[string]any) (any, error) {
			return time.Now().Format(time.RFC3339), nil
		},
	})

	response, err := helperAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{Role: "user", Content: "What time is it now?"},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nFinal response:")
	fmt.Println(response.Messages[len(response.Messages)-1].Content)
}
