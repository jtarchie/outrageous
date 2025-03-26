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

	a.Expect(response.Messages).To(gomega.HaveLen(2))
	a.Expect(response.Agent.Name()).To(gomega.Equal("HelpfulAgent"))

	result, err := assert.Agent(
		fmt.Sprintf("This messages should be a positive greeting to a user: %s", response.Messages[1].Content),
	)
	a.Expect(err).NotTo(gomega.HaveOccurred())
	a.Expect(result.Status).To(gomega.Equal(assert.Success), result.Explanation)
}
