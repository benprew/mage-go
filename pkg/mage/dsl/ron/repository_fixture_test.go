package ron

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
)

type repositoryRecordKey struct {
	path, source string
}

type repositoryRecordCache struct {
	mu      sync.Mutex
	files   fs.FS
	records map[repositoryRecordKey]map[string]Node
}

var repositoryFixtures = repositoryRecordCache{files: os.DirFS("../../../../cards/mtgish")}

func (c *repositoryRecordCache) record(path, source, name string) (Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := repositoryRecordKey{path: path, source: source}
	records, ok := c.records[key]
	if !ok {
		data, err := fs.ReadFile(c.files, path)
		if err != nil {
			return Node{}, err
		}
		nodes, err := Parse(strings.NewReader(string(data)), source)
		if err != nil {
			return Node{}, err
		}
		records = make(map[string]Node, len(nodes))
		for _, node := range nodes {
			cardName := node.Fields["Name"].String
			if _, exists := records[cardName]; !exists {
				records[cardName] = node
			}
		}
		if c.records == nil {
			c.records = make(map[repositoryRecordKey]map[string]Node)
		}
		c.records[key] = records
	}
	node, ok := records[name]
	if !ok {
		return Node{}, fmt.Errorf("missing card %s in %s", name, path)
	}
	return cloneRepositoryNode(node), nil
}

// Tests can change subrules, so each lookup needs a separate syntax tree.
func cloneRepositoryNode(node Node) Node {
	if node.Args != nil {
		args := make([]Node, len(node.Args))
		for i, arg := range node.Args {
			args[i] = cloneRepositoryNode(arg)
		}
		node.Args = args
	}
	if node.Fields != nil {
		fields := make(map[string]Node, len(node.Fields))
		for name, field := range node.Fields {
			fields[name] = cloneRepositoryNode(field)
		}
		node.Fields = fields
	}
	return node
}

func repositoryRecord(t *testing.T, set, name, source string) Node {
	t.Helper()
	node, err := repositoryFixtures.record(set+".lines.ron", source, name)
	if err != nil {
		t.Fatal(err)
	}
	return node
}
