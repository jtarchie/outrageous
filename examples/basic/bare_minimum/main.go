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
	err := examples.Run(BasicExample)
	if err != nil {
		log.Fatal(err)
	}
}

func BasicExample() (*agent.Response, error) {
	helpfulAgent := agent.New(
		"Agent",
		"You are a helpful agent.",
	)

	response, err := helpfulAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{Role: "user", Content: "Hi"},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to run: %w", err)
	}

	return response, nil
}
