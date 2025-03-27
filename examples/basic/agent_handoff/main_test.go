package main

import (
	"fmt"
	"testing"

	"github.com/jtarchie/outrageous/assert"
	"github.com/onsi/gomega"
)

func TestAgentHandoff(t *testing.T) {
	a := gomega.NewGomegaWithT(t)

	response, err := AgentHandoff()
	a.Expect(err).NotTo(gomega.HaveOccurred())

	a.Expect(response.Messages).To(gomega.HaveLen(5))
	a.Expect(response.Agent.Name()).To(gomega.Equal("SpanishAgent"))

	message := response.Messages[len(response.Messages)-1].Content
	result, err := assert.Agent(
		fmt.Sprintf("This message should be in Spanish: %q", message),
	)
	a.Expect(err).NotTo(gomega.HaveOccurred())
	a.Expect(result.Status).To(gomega.Equal(assert.Success), fmt.Sprintf("explain: %s, message: %s", result.Explanation, message))
}
