package dsl_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

type typeImporter map[string]*types.Package

func (i typeImporter) Import(path string) (*types.Package, error) { return i[path], nil }

func TestClosedExpressionRoles(t *testing.T) {
	loaded, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps}, "github.com/benprew/mage-go/pkg/mage/dsl")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(loaded) > 0 {
		t.Fatal("load DSL types")
	}
	importer := typeImporter{}
	packages.Visit(loaded, nil, func(p *packages.Package) { importer[p.PkgPath] = p.Types })
	files, err := filepath.Glob("testdata/compilefail/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, want, ok := strings.Cut(string(source), "// ERROR ")
			if !ok {
				t.Fatal("missing expected diagnostic")
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, source, 0)
			if err != nil {
				t.Fatal(err)
			}
			conf := types.Config{Importer: importer}
			_, err = conf.Check("fixture", fset, []*ast.File{f}, nil)
			if err == nil || !strings.Contains(err.Error(), strings.TrimSpace(want)) {
				t.Fatalf("error = %v; want %s", err, want)
			}
		})
	}
}
