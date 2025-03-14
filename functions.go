package outrageous

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Function struct {
	Name        string
	Description string
	Parameters  *jsonschema.Definition
	Func        func(ctx context.Context, params map[string]any) (any, error)
}
type Functions []Function

func (f *Functions) Add(functions ...Function) {
	*f = append(*f, functions...)
}

func (f *Functions) Get(name string) (Function, bool) {
	for _, function := range *f {
		if function.Name == name || toName(function.Name) == name {
			return function, true
		}
	}
	return Function{}, false
}

func toName(name string) string {
	// name Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a maximum length of 64
	return regexp.MustCompile(`[^a-zA-Z0-9_.-]`).ReplaceAllString(name, "")
}

func (f Functions) AsTools() []openai.Tool {
	tools := []openai.Tool{}
	for _, function := range f {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        toName(function.Name),
				Description: function.Description,
				Parameters:  function.Parameters,
				Strict:      true,
			},
		})
	}
	return tools
}

type Caller interface {
	Call(ctx context.Context) (any, error)
}

func MustWrapStruct(description string, s Caller) Function {
	function, err := WrapStruct(description, s)
	if err != nil {
		panic(fmt.Sprintf("could not wrap struct: %v", err))
	}
	return function
}

func WrapStruct(description string, s Caller) (Function, error) {
	structName := reflect.TypeOf(s).Name()
	instance := reflect.New(reflect.TypeOf(s)).Interface()

	schema, err := jsonschema.GenerateSchemaForType(instance)
	if err != nil {
		return Function{}, fmt.Errorf("could not generate schema: %w", err)
	}

	slog.Debug("function.schema", "name", structName, "schema", schema, "instance", instance)

	return Function{
		Name:        structName,
		Description: description,
		Parameters:  schema,
		Func: func(ctx context.Context, params map[string]any) (any, error) {
			slog.Debug("function.call", "name", structName, "params", params)

			// Create a new instance of the struct
			instance := reflect.New(reflect.TypeOf(s)).Interface()

			// Populate the struct with the parameters
			for key, value := range params {
				field := reflect.ValueOf(instance).Elem().FieldByName(key)
				slog.Debug("function.call", "name", structName, "key", key, "value", value, "field", field)
				if !field.IsValid() {
					// find field by json tag
					for i := 0; i < reflect.TypeOf(s).NumField(); i++ {
						fieldType := reflect.TypeOf(s).Field(i)
						jsonTag := fieldType.Tag.Get("json")
						slog.Debug("function.call", "name", structName, "jsonTag", jsonTag, "key", key)
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
