package outrageous_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"testing"

	"github.com/onsi/gomega"
)

func TestRead(t *testing.T) {
	a := gomega.NewGomegaWithT(t)
	
	contents, err := os.ReadFile("README.md")
	a.Expect(err).NotTo(gomega.HaveOccurred())

	codeBlocks := regexp.MustCompile("(?ms)```go(.*?)```").FindAllStringSubmatch(string(contents), -1)
	a.Expect(len(codeBlocks)).To(gomega.BeNumerically(">", 0))

	for index, codeBlock := range codeBlocks {
		code := codeBlock[1]
		_, err = parser.ParseFile(token.NewFileSet(), "main.go", code, parser.AllErrors)
		a.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Code block %d is not valid Go code: %s", index, code))
	}
}	