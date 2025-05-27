package agent_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jtarchie/outrageous/agent"
	"github.com/onsi/gomega"
)

// Test structs that implement the Caller interface
type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (ts *TestStruct) Call(ctx context.Context) (any, error) {
	return map[string]interface{}{
		"message": "Hello " + ts.Name,
		"age":     ts.Age,
	}, nil
}

type ErrorStruct struct {
	Value string `json:"value"`
}

func (es *ErrorStruct) Call(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("test error: %s", es.Value)
}

type ComplexStruct struct {
	ID       int      `json:"id"`
	Tags     []string `json:"tags"`
	Metadata struct {
		Author string `json:"author"`
		Count  int    `json:"count"`
	} `json:"metadata"`
}

func (cs *ComplexStruct) Call(ctx context.Context) (any, error) {
	return map[string]interface{}{
		"processed": true,
		"id":        cs.ID,
		"tagCount":  len(cs.Tags),
		"author":    cs.Metadata.Author,
	}, nil
}

func TestToolWrapStruct(t *testing.T) {
	t.Run("should wrap struct successfully", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Test tool description", &TestStruct{})

		g.Expect(err).To(gomega.BeNil())
		g.Expect(tool.Name).To(gomega.Equal("TestStruct"))
		g.Expect(tool.Description).To(gomega.Equal("Test tool description"))
		g.Expect(tool.Parameters).ToNot(gomega.BeNil())
		g.Expect(tool.Func).ToNot(gomega.BeNil())
	})

	t.Run("should work with non-pointer struct", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Non-pointer test", &TestStruct{})

		g.Expect(err).To(gomega.BeNil())
		g.Expect(tool.Name).To(gomega.Equal("TestStruct"))
	})

	t.Run("should generate correct schema", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Schema test", &TestStruct{})

		g.Expect(err).To(gomega.BeNil())
		g.Expect(tool.Parameters.Type).To(gomega.BeEquivalentTo("object"))
		g.Expect(tool.Parameters.Properties).To(gomega.HaveKey("name"))
		g.Expect(tool.Parameters.Properties).To(gomega.HaveKey("age"))
	})

	t.Run("should execute function with correct parameters", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Execution test", &TestStruct{})
		g.Expect(err).To(gomega.BeNil())

		params := map[string]any{
			"name": "Alice",
			"age":  30,
		}

		result, err := tool.Func(context.Background(), params)

		g.Expect(err).To(gomega.BeNil())
		g.Expect(result).To(gomega.Equal(map[string]interface{}{
			"message": "Hello Alice",
			"age":     30,
		}))
	})

	t.Run("should handle function errors", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Error test", &ErrorStruct{})
		g.Expect(err).To(gomega.BeNil())

		params := map[string]any{
			"value": "test error message",
		}

		result, err := tool.Func(context.Background(), params)

		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err.Error()).To(gomega.ContainSubstring("test error: test error message"))
		g.Expect(result).To(gomega.BeNil())
	})

	t.Run("should handle complex nested structures", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Complex test", &ComplexStruct{})
		g.Expect(err).To(gomega.BeNil())

		params := map[string]any{
			"id":   123,
			"tags": []string{"go", "test", "agent"},
			"metadata": map[string]any{
				"author": "test-author",
				"count":  5,
			},
		}

		result, err := tool.Func(context.Background(), params)

		g.Expect(err).To(gomega.BeNil())
		resultMap := result.(map[string]interface{})
		g.Expect(resultMap["processed"]).To(gomega.BeTrue())
		g.Expect(resultMap["id"]).To(gomega.Equal(123))
		g.Expect(resultMap["tagCount"]).To(gomega.Equal(3))
		g.Expect(resultMap["author"]).To(gomega.Equal("test-author"))
	})

	t.Run("should handle invalid JSON parameters", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Invalid params test", &TestStruct{})
		g.Expect(err).To(gomega.BeNil())

		// Invalid params that can't be unmarshaled to TestStruct
		params := map[string]any{
			"age": "not a number", // age should be int
		}

		result, err := tool.Func(context.Background(), params)

		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err.Error()).To(gomega.ContainSubstring("could not unmarshal params"))
		g.Expect(result).To(gomega.BeNil())
	})

	t.Run("should persist setup across multiple calls", func(t *testing.T) {
		g := gomega.NewWithT(t)

		tool, err := agent.WrapStruct("Persistence test", &TestStruct{})
		g.Expect(err).To(gomega.BeNil())

		// First call
		params1 := map[string]any{
			"name": "First",
			"age":  25,
		}
		result1, err := tool.Func(context.Background(), params1)
		g.Expect(err).To(gomega.BeNil())

		// Second call with different parameters
		params2 := map[string]any{
			"name": "Second",
			"age":  35,
		}
		result2, err := tool.Func(context.Background(), params2)
		g.Expect(err).To(gomega.BeNil())

		// Verify both calls worked independently
		g.Expect(result1).To(gomega.Equal(map[string]interface{}{
			"message": "Hello First",
			"age":     25,
		}))
		g.Expect(result2).To(gomega.Equal(map[string]interface{}{
			"message": "Hello Second",
			"age":     35,
		}))
	})
}
