package agent

import (
	"context"
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
		if tool.Name == name || toName(tool.Name) == toName(name) {
			return tool, true
		}
	}
	return Tool{}, false
}

func toName(name string) string {
	// name Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a maximum length of 64
	return regexp.MustCompile(`[^a-zA-Z0-9_.-]`).ReplaceAllString(name, "")
}

func (f Tools) AsTools() []openai.Tool {
	tools := []openai.Tool{}
	for _, tool := range f {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        toName(tool.Name),
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

			// Populate the struct with the parameters
			for key, value := range params {
				field := reflect.ValueOf(instance).Elem().FieldByName(key)
				slog.Debug("tool.call", "name", structName, "key", key, "value", value, "field", field)
				if !field.IsValid() {
					// find field by json tag
					for i := 0; i < reflect.TypeOf(s).NumField(); i++ {
						fieldType := reflect.TypeOf(s).Field(i)
						jsonTag := fieldType.Tag.Get("json")
						slog.Debug("tool.call", "name", structName, "jsonTag", jsonTag, "key", key)
						if jsonTag == key {
							field = reflect.ValueOf(instance).Elem().Field(i)
							break
						}
					}
				}

				if field.IsValid() && field.CanSet() {
					field.Set(reflect.ValueOf(value))
				}
			}

			slog.Debug("tool.call", "name", structName, "instance", instance)

			// Call the method on the struct
			method := reflect.ValueOf(instance).MethodByName("Call")
			if !method.IsValid() {
				return nil, fmt.Errorf("method Call not found on struct %s", structName)
			}

			values := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			// return the first value and error
			err = nil
			if !values[1].IsNil() {
				err = values[1].Interface().(error)
			}

			return values[0].Interface(), err
		},
	}, nil
}
