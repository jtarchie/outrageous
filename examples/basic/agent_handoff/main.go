package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/examples"
)

func main() {
	// examples.Run is only used to run examples
	// it allows for testing of the examples
	err := examples.Run(AgentHandoff)
	if err != nil {
		log.Fatal(err)
	}
}

func AgentHandoff() (*agent.Response, error) {
	englishAgent := agent.New(
		"English Agent",
		"You only speak English",
	)

	spanishAgent := agent.New(
		"Spanish Agent",
		"You only speak Spanish",
	)

	englishAgent.Tools.Add(spanishAgent.AsTool("Transfer spanish speaking users immediately"))

	response, err := englishAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{Role: "user", Content: "Hola. ¿Como estás?"},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run: %w", err)
	}

	return response, nil
}
