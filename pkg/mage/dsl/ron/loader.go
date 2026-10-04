package ron

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
)

// Issue records a card that cannot be loaded without a rule change.
type Issue struct {
	Card    string   `json:"card,omitempty"`
	Source  Position `json:"source"`
	Message string   `json:"message"`
}

// LoadError contains all card failures found in a collection.
type LoadError struct {
	Records  int     `json:"records"`
	Compiled int     `json:"compiled"`
	Issues   []Issue `json:"issues"`
}

func (e *LoadError) Error() string {
	var out strings.Builder
	fmt.Fprintf(&out, "RON collection rejected: %d issue(s), %d/%d records compiled", len(e.Issues), e.Compiled, e.Records)
	for _, issue := range e.Issues {
		fmt.Fprintf(&out, "\n%s: %s: %s", issue.Source, issue.Card, issue.Message)
	}
	return out.String()
}

// LoadInto replaces all factories only after the complete collection compiles.
func LoadInto(registry *mage.Registry, files fs.FS, pattern string) error {
	if registry == nil {
		return fmt.Errorf("RON load requires a registry")
	}
	replacement, err := LoadFS(files, pattern)
	if err != nil {
		return err
	}
	registry.ReplaceWith(replacement)
	return nil
}

// LoadFS builds a new registry only if every card compiles.
// It does not change the process registry or reuse registered Go factories.
func LoadFS(files fs.FS, pattern string) (*mage.Registry, error) {
	paths, err := fs.Glob(files, pattern)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no RON files match %q", pattern)
	}
	report := &LoadError{}
	factories := make(map[string]mage.CardFactory)
	definitions := make(map[string]string)
	positions := make(map[string]Position)
	for _, path := range paths {
		data, err := fs.ReadFile(files, path)
		if err != nil {
			report.Issues = append(report.Issues, Issue{Source: Position{File: path}, Message: err.Error()})
			continue
		}
		nodes, err := Parse(strings.NewReader(string(data)), path)
		if err != nil {
			report.Issues = append(report.Issues, Issue{Source: Position{File: path}, Message: err.Error()})
			continue
		}
		if len(nodes) == 0 {
			report.Issues = append(report.Issues, Issue{Source: Position{File: path}, Message: "empty card file"})
		}
		for _, node := range nodes {
			report.Records++
			name := node.Fields["Name"].String
			factory, err := Compile(node)
			if err != nil {
				report.Issues = append(report.Issues, Issue{Card: name, Source: node.Position, Message: err.Error()})
				continue
			}
			key := definitionKey(node)
			if previous, ok := definitions[name]; ok && previous != key {
				report.Issues = append(report.Issues, Issue{Card: name, Source: node.Position, Message: fmt.Sprintf("conflicting definition; first definition at %s", positions[name])})
				continue
			}
			report.Compiled++
			if _, exists := definitions[name]; !exists {
				definitions[name], factories[name], positions[name] = key, factory, node.Position
			}
		}
	}
	if len(report.Issues) != 0 {
		return nil, report
	}
	registry := mage.NewRegistry()
	for name, factory := range factories {
		registry.Register(name, factory)
	}
	return registry, nil
}

func definitionKey(node Node) string {
	node = withoutPositions(node)
	data, _ := json.Marshal(node)
	return string(data)
}

func withoutPositions(node Node) Node {
	nodeCopy := node
	nodeCopy.Position = Position{}
	nodeCopy.Args = make([]Node, len(node.Args))
	for i, arg := range node.Args {
		nodeCopy.Args[i] = withoutPositions(arg)
	}
	if node.Fields != nil {
		nodeCopy.Fields = make(map[string]Node, len(node.Fields))
		for key, field := range node.Fields {
			nodeCopy.Fields[key] = withoutPositions(field)
		}
	}
	return nodeCopy
}
