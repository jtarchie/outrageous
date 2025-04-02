package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/examples"
)

type GetWeather struct {
	Location string `json:"location"`
}

func (g GetWeather) Call(ctx context.Context) (any, error) {
	if strings.Contains(strings.ToLower(g.Location), "nyc") {
		return `{"temp":32, "unit":"F"}`, nil
	}

	return `{"temp":67, "unit":"F"}`, nil
}

func main() {
	// examples.Run is only used to run examples
	// it allows for testing of the examples
	err := examples.Run(FunctionCalling)
	if err != nil {
		log.Fatal(err)
	}
}

func FunctionCalling() (*agent.Response, error) {
	helpfulAgent := agent.New(
		"FunctionAgent",
		"You are a helpful agent.",
	)

	fn, err := agent.WrapStruct("returns the current weather", GetWeather{})
	if err != nil {
		log.Fatal(err)
	}

	helpfulAgent.Tools.Add(fn)

	response, err := helpfulAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{Role: "user", Content: "What's the weather in NYC?"},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to run: %w", err)
	}

	return response, nil
}
