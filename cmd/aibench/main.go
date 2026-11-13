// Command aibench compares the working tree AI with committed HEAD.
package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const moduleImport = "github.com/benprew/mage-go"
const decisionImport = moduleImport + "/pkg/mage/interactive"
const historicalImport = moduleImport + "/internal/benchmarkhead"

func historicalSource(source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "snapshot.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		if path == decisionImport || strings.HasPrefix(path, decisionImport+"/") {
			spec.Path.Value = strconv.Quote(historicalImport + strings.TrimPrefix(path, decisionImport))
		} else if strings.HasPrefix(path, moduleImport+"/") && path != moduleImport+"/pkg/mage" && path != moduleImport+"/pkg/mage/core" {
			return nil, fmt.Errorf("historical AI has an unisolated dependency: %s", path)
		}
	}
	var out bytes.Buffer
	err = format.Node(&out, fset, f)
	return out.Bytes(), err
}

func git(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w", args, err)
	}
	return out, nil
}

func writeFile(root, name string, data []byte, mode os.FileMode) error {
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func committedGamePath(name string) bool {
	return strings.HasPrefix(name, "cards/") || (strings.HasPrefix(name, "pkg/mage/") && !strings.HasPrefix(name, "pkg/mage/interactive/"))
}

func copyWorkingTree(root, dest string) error {
	files, err := git(root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for name := range strings.SplitSeq(string(files), "\x00") {
		if name == "" || seen[name] || strings.HasSuffix(name, ".log") || strings.HasPrefix(name, "internal/benchmarkhead/") || committedGamePath(name) {
			continue
		}
		seen[name] = true
		info, err := os.Lstat(filepath.Join(root, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			info, err = os.Stat(filepath.Join(root, name))
			if err != nil {
				return err
			}
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot snapshot nonregular file %s", name)
		}
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if err := writeFile(dest, name, data, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func snapshotHEAD(root, dest, revision string) error {
	archive, err := git(root, "archive", revision, "pkg/mage", "cards")
	if err != nil {
		return err
	}
	reader := tar.NewReader(bytes.NewReader(archive))
	count := 0
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		historical := strings.HasPrefix(header.Name, "pkg/mage/interactive/")
		if historical && (!strings.HasSuffix(header.Name, ".go") || strings.HasSuffix(header.Name, "_test.go")) {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		name := header.Name
		if historical {
			data, err = historicalSource(data)
			if err != nil {
				return fmt.Errorf("%s: %w", header.Name, err)
			}
			name = strings.Replace(header.Name, "pkg/mage/interactive", "internal/benchmarkhead", 1)
			count++
		}
		if err := writeFile(dest, name, data, 0o644); err != nil {
			return err
		}
	}
	if count == 0 {
		return errors.New("HEAD has no AI source")
	}
	return nil
}

func run() error {
	rootBytes, err := git("", "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	root := strings.TrimSpace(string(rootBytes))
	revisionBytes, err := git(root, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return err
	}
	revision := strings.TrimSpace(string(revisionBytes))
	dest, err := os.MkdirTemp("", "mage-aibench-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dest)
	if err := copyWorkingTree(root, dest); err != nil {
		return err
	}
	if err := snapshotHEAD(root, dest, revision); err != nil {
		return err
	}
	if err := writeFile(dest, "pkg/mage/interactive/ai/heuristic/historical_head_test.go", []byte(comparisonTest), 0o644); err != nil {
		return err
	}
	fmt.Printf("Historical AI: committed HEAD %s\nCandidate AI: working tree\nShared: committed HEAD rules engine and cards\n", revision)
	cmd := exec.Command("go", "test", "./pkg/mage/interactive/ai/heuristic", "-run", "^TestSimulation_HistoricalHEAD$", "-count=1", "-timeout=30m", "-v")
	cmd.Dir = dest
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const comparisonTest = `package heuristic

import (
 "testing"
 "github.com/benprew/mage-go/internal/tui"
 oldinteractive "github.com/benprew/mage-go/internal/benchmarkhead"
 oldai "github.com/benprew/mage-go/internal/benchmarkhead/ai"
 oldheuristic "github.com/benprew/mage-go/internal/benchmarkhead/ai/heuristic"
 "github.com/benprew/mage-go/pkg/mage"
 "github.com/benprew/mage-go/pkg/mage/interactive"
 "github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

type historicalPlayer struct { *oldai.AIPlayer }

func (p *historicalPlayer) GetPriorityAction(g *mage.Game, lands int, main bool) interactive.PriorityAction {
 a := p.AIPlayer.GetPriorityAction(g, lands, main)
 kind := interactive.ActionPass
 switch a.Type {
 case oldinteractive.ActionPlayLand: kind = interactive.ActionPlayLand
 case oldinteractive.ActionCastSpell: kind = interactive.ActionCastSpell
 case oldinteractive.ActionActivateAbility: kind = interactive.ActionActivateAbility
 case oldinteractive.ActionPass:
 default: panic("unsupported historical priority action")
 }
 return interactive.PriorityAction{Type: kind, CardID: a.CardID, Targets: a.Targets, XValue: a.XValue, PermanentID: a.PermanentID, AbilityIndex: a.AbilityIndex}
}

func historicalFactory(name string, deck []tui.DeckEntry) simulationPlayer {
 entries := make([]oldai.DeckCard, len(deck))
 for i, e := range deck { entries[i] = oldai.DeckCard{Name: e.Name, Count: e.Count} }
 return &historicalPlayer{oldai.NewAIPlayer(name, oldheuristic.New(oldai.InferPersonalityFromDeck(entries)))}
}

func TestSimulation_HistoricalHEAD(t *testing.T) {
 for _, deck := range simulationArchetypes(t) {
  entries := make([]oldai.DeckCard, len(deck.Entries))
  for i, e := range deck.Entries { entries[i] = oldai.DeckCard{Name: e.Name, Count: e.Count} }
  t.Logf("%s: current personality %s; HEAD personality %s", deck.Name, ai.InferPersonalityFromDeck(aiDeckEntries(deck.Entries)).Name, oldai.InferPersonalityFromDeck(entries).Name)
 }
 t.Run("Control", func(t *testing.T) { runSimulationGames(t, "HEAD vs HEAD control", historicalFactory, historicalFactory) })
 t.Run("Candidate", func(t *testing.T) {
  runSimulationGames(t, "Working tree vs committed HEAD", func(name string, deck []tui.DeckEntry) simulationPlayer {
   return ai.NewAIPlayer(name, New(ai.InferPersonalityFromDeck(aiDeckEntries(deck))))
  }, historicalFactory)
 })
}
`
