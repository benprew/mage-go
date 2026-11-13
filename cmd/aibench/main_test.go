package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHistoricalSourceIsolatesDecisionImports(t *testing.T) {
	source := []byte(`package heuristic
import (
 "github.com/benprew/mage-go/pkg/mage"
 "github.com/benprew/mage-go/pkg/mage/interactive"
 "github.com/benprew/mage-go/pkg/mage/interactive/ai"
 "github.com/benprew/mage-go/pkg/mage/interactive/ai/combatsolver"
 "github.com/benprew/mage-go/pkg/mage/interactive/eval"
)
`)
	got, err := historicalSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), `"github.com/benprew/mage-go/pkg/mage/interactive`) {
		t.Fatalf("historical code can still import current decision code:\n%s", got)
	}
	for _, suffix := range []string{"", "/ai", "/ai/combatsolver", "/eval"} {
		if !strings.Contains(string(got), `"`+historicalImport+suffix+`"`) {
			t.Errorf("missing historical import %s", suffix)
		}
	}
	if !strings.Contains(string(got), `"github.com/benprew/mage-go/pkg/mage"`) {
		t.Fatal("rules engine import changed")
	}
}

func TestHistoricalSourceRejectsUnisolatedDependency(t *testing.T) {
	_, err := historicalSource([]byte(`package ai; import "github.com/benprew/mage-go/internal/newpolicy"`))
	if err == nil {
		t.Fatal("unisolated decision dependency must fail")
	}
}

func TestSnapshotHEADHasNoCurrentDecisionImports(t *testing.T) {
	rootBytes, err := git("", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	root := strings.TrimSpace(string(rootBytes))
	revision, err := git(root, "rev-parse", "HEAD^{commit}")
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := snapshotHEAD(root, dest, strings.TrimSpace(string(revision))); err != nil {
		t.Fatal(err)
	}
	files := 0
	err = filepath.WalkDir(filepath.Join(dest, "internal/benchmarkhead"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, data, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range f.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if importPath == decisionImport || strings.HasPrefix(importPath, decisionImport+"/") {
				t.Errorf("%s still imports current decision code %s", path, importPath)
			}
		}
		files++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("snapshot contains no historical source")
	}
	for _, name := range []string{"ai/ai.go", "ai/personality.go", "ai/deck_personality.go", "ai/heuristic/heuristic.go", "ai/combatsolver/solver.go", "eval/eval.go"} {
		if _, err := os.Stat(filepath.Join(dest, "internal/benchmarkhead", name)); err != nil {
			t.Errorf("missing historical dependency %s: %v", name, err)
		}
	}
}

func TestCommittedGamePaths(t *testing.T) {
	for name, want := range map[string]bool{
		"pkg/mage/mode_heuristic.go":        true,
		"pkg/mage/core/types.go":            true,
		"cards/limited/spells.go":           true,
		"pkg/mage/interactive/ai/ai.go":     false,
		"pkg/mage/interactive/eval/eval.go": false,
		"internal/tui/decks.go":             false,
	} {
		if got := committedGamePath(name); got != want {
			t.Errorf("%s: committed=%v, want %v", name, got, want)
		}
	}
}
