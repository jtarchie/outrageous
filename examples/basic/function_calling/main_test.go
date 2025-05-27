package main

import (
	"fmt"
	"testing"

	"github.com/jtarchie/outrageous/assert"
	"github.com/onsi/gomega"
)

func TestFunctionCalling(t *testing.T) {
	a := gomega.NewGomegaWithT(t)

	response, err := FunctionCalling()
	a.Expect(err).NotTo(gomega.HaveOccurred())

	a.Expect(response.Messages).To(gomega.HaveLen(5))
	a.Expect(response.Agent.Name()).To(gomega.Equal("FunctionAgent"))

	a.Expect(response.Messages[3].Role).To(gomega.Equal("tool"))
	a.Expect(response.Messages[3].Name).To(gomega.Equal("get_weather"))

	a.Expect(response.Messages[4].Role).To(gomega.Equal("assistant"))
	a.Expect(response.Messages[4].Name).To(gomega.Equal("FunctionAgent"))

	message := response.Messages[len(response.Messages)-1].Content
	result, err := assert.Agent(
		fmt.Sprintf("This message should be about weather: %q", message),
	)
	a.Expect(err).NotTo(gomega.HaveOccurred())
	a.Expect(result.Status).To(gomega.Equal(assert.Success), fmt.Sprintf("explain: %s, message: %s", result.Explanation, message))
}
