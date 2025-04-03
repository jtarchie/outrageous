package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Tool struct {
	Name        string
	Description string
	Parameters  *jsonschema.Definition
	Func        func(ctx context.Context, params map[string]any) (any, error)
}
type Tools []Tool

func (f *Tools) Add(tools ...Tool) {
	for _, tool := range tools {
		slog.Debug("agent.tools.add", "name", tool.Name, "description", tool.Description)
		*f = append(*f, tool)
	}
}

func (f *Tools) Get(name string) (Tool, bool) {
	for _, tool := range *f {
		if tool.Name == name || toFormattedName(tool.Name) == toFormattedName(name) {
			return tool, true
		}
	}
	return Tool{}, false
}

func toFormattedName(name string) string {
	// name Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a maximum length of 64
	return regexp.MustCompile(`[^a-zA-Z0-9_.-]`).ReplaceAllString(name, "")
}

func (f Tools) AsTools() []openai.Tool {
	tools := []openai.Tool{}
	for _, tool := range f {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        toFormattedName(tool.Name),
				Description: tool.Description,
				Parameters:  tool.Parameters,
				Strict:      true,
			},
		})
	}
	return tools
}

type Caller interface {
	Call(ctx context.Context) (any, error)
}

func MustWrapStruct(description string, s Caller) Tool {
	tool, err := WrapStruct(description, s)
	if err != nil {
		panic(fmt.Sprintf("could not wrap struct: %v", err))
	}
	return tool
}

func WrapStruct(description string, s Caller) (Tool, error) {
	structName := reflect.TypeOf(s).Name()
	instance := reflect.New(reflect.TypeOf(s)).Interface()

	schema, err := jsonschema.GenerateSchemaForType(instance)
	if err != nil {
		return Tool{}, fmt.Errorf("could not generate schema: %w", err)
	}

	slog.Debug("tool.schema", "name", structName, "schema", schema, "instance", instance)

	return Tool{
		Name:        structName,
		Description: description,
		Parameters:  schema,
		Func: func(ctx context.Context, params map[string]any) (any, error) {
			slog.Debug("tool.call", "name", structName, "params", params)

			// Create a new instance of the struct
			instance := reflect.New(reflect.TypeOf(s)).Interface()

			contents, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("could not marshal params: %w", err)
			}

			err = json.Unmarshal(contents, &instance)
			if err != nil {
				return nil, fmt.Errorf("could not unmarshal params: %w", err)
			}

			slog.Debug("tool.call", "name", structName, "instance", instance)

			// Call the method on the struct
			caller := instance.(Caller)
			result, err := caller.Call(ctx)
			if err != nil {
				return nil, fmt.Errorf("could not call method: %w", err)
			}
			slog.Debug("tool.call", "name", structName, "result", result)

			return result, nil
		},
	}, nil
}
