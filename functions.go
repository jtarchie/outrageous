package outrageous

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Function struct {
	Name        string
	Description string
	Parameters  jsonschema.Definition
	Func        func(ctx context.Context, params map[string]any) (any, error)
}
type Functions []Function

func (f *Functions) Add(function Function) {
	*f = append(*f, function)
}

func (f *Functions) Get(name string) (Function, bool) {
	for _, function := range *f {
		if function.Name == name {
			return function, true
		}
	}
	return Function{}, false
}

func (f Functions) AsTools() []openai.Tool {
	tools := make([]openai.Tool, len(f))
	for i, function := range f {
		tools[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        function.Name,
				Description: function.Description,
				Parameters:  function.Parameters,
			},
		}
	}
	return tools

}
