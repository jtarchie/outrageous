package outrageous_test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
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
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "main.go", code, parser.AllErrors)
		a.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Code block %d is not valid Go code: %s", index, code))

		conf := types.Config{
			Importer: importer.ForCompiler(fset, "source", nil),
		}
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
		}
		_, err = conf.Check("main", fset, []*ast.File{node}, info)
		a.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Code block %d failed type checking: %s", index, code))
	}
}	