package ron

import (
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

type repositoryCountingFS struct {
	fs.FS
	reads int
}

func (f *repositoryCountingFS) ReadFile(name string) ([]byte, error) {
	f.reads++
	return fs.ReadFile(f.FS, name)
}

func TestRepositoryRecordCacheIsolation(t *testing.T) {
	files := &repositoryCountingFS{FS: fstest.MapFS{"set.ron": {Data: []byte(testBolt)}}}
	cache := repositoryRecordCache{files: files}
	nodes, err := Parse(strings.NewReader(testBolt), "source.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := nodes[0]
	first, err := cache.record("set.ron", "source.ron", "RON Bolt")
	if err != nil || !reflect.DeepEqual(first, want) {
		t.Fatalf("first record: %v", err)
	}
	rules := first.Fields["Rules"]
	rules.Args[0].Args[0].Args[0].Args[0].Name = "Changed"
	first.Fields["Name"] = Node{Kind: String, String: "Changed"}
	delete(first.Fields["Typeline"].Fields, "Cardtypes")
	second, err := cache.record("set.ron", "source.ron", "RON Bolt")
	if err != nil || !reflect.DeepEqual(second, want) {
		t.Fatalf("cached record changed: %v", err)
	}
	if files.reads != 1 {
		t.Fatalf("file read %d times, want 1", files.reads)
	}
	if _, err := cache.record("set.ron", "source.ron", "Missing"); err == nil {
		t.Fatal("missing card accepted")
	}
	other, err := cache.record("set.ron", "other.ron", "RON Bolt")
	if err != nil || other.Position.File != "other.ron" {
		t.Fatalf("source file changed: %v", err)
	}
}
