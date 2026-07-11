package mage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPermanentControllerWritesAreCentralized(t *testing.T) {
	allowed := map[string]bool{
		"card.go":           true,
		"clone.go":          true,
		"effect_control.go": true,
		"effect_manager.go": true,
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || allowed[name] {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assign.Lhs {
				selector, ok := lhs.(*ast.SelectorExpr)
				if ok && (selector.Sel.Name == "computedController" || selector.Sel.Name == "baseController") {
					pos := fset.Position(selector.Pos())
					t.Errorf("controller write outside construction, cloning, or Layer 2 reconciliation: %s:%d", name, pos.Line)
				}
			}
			return true
		})
	}
}
