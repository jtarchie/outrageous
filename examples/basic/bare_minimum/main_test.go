package main

import (
	"fmt"
	"testing"

	"github.com/jtarchie/outrageous/assert"
	"github.com/onsi/gomega"
)

func TestBareMinimum(t *testing.T) {
	a := gomega.NewGomegaWithT(t)

	response, err := BasicExample()
	a.Expect(err).NotTo(gomega.HaveOccurred())

	a.Expect(response.Messages).To(gomega.HaveLen(3))
	a.Expect(response.Agent.Name()).To(gomega.Equal("helpful_agent"))

	message := response.Messages[1].Content
	result, err := assert.Agent(
		fmt.Sprintf("This should be a greeting: %q", message),
	)
	a.Expect(err).NotTo(gomega.HaveOccurred())
	a.Expect(result.Status).To(gomega.Equal(assert.Success), fmt.Sprintf("explain: %s, message: %s", result.Explanation, message))
}
