package assert

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jtarchie/outrageous/agent"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Status string

const (
	Success Status = "success"
	Failure Status = "failure"
)

//go:embed prompt.md
var prompt string

type Result struct {
	Explanation string `json:"explanation" description:"The explanation of the assertion. This is a human readable explanation of the assertion. It should be a single sentence." required:"true"`
	Status      Status `json:"status" description:"The status of the assertion. Either success or failure" required:"true" enum:"success,failure"`
}

func Agent(assertion string, opts ...agent.AgentOption) (Result, error) {
	assertionAgent := agent.New(
		"Assertion Agent",
		prompt,
		opts...,
	)

	schema, err := jsonschema.GenerateSchemaForType(Result{})
	if err != nil {
		return Result{}, fmt.Errorf("could not generate schema: %w", err)
	}

	prop := schema.Properties["status"]
	schema.Properties["status"] = prop

	var result Result
	assertionAgent.Tools.Add(agent.Tool{
		Name:        "assertion",
		Description: "Function that is called when the assertion is complete",
		Parameters:  schema,
		Func: func(ctx context.Context, params map[string]any) (any, error) {
			// this is the function that is called when the assertion is complete
			// it will be called with the status of the assertion
			if state, ok := params["status"].(string); ok {
				result.Status = Status(state)
			} else {
				return nil, fmt.Errorf("could not parse status: %v", params["status"])
			}

			if explanation, ok := params["explanation"].(string); ok {
				result.Explanation = explanation
			} else {
				return nil, fmt.Errorf("could not parse explanation: %v", params["explanation"])
			}

			return result.Status, nil
		},
	})

	for range 3 {
		_, err = assertionAgent.Run(
			context.Background(),
			agent.Messages{
				agent.Message{Role: "user", Content: assertion},
			},
		)
		if err != nil {
			return Result{}, fmt.Errorf("failed to run: %w", err)
		}

		if result.Status == "success" || result.Status == "failure" {
			break
		}
	}

	return result, nil
}
