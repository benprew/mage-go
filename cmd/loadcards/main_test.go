package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/dsl/ron"
)

func TestInventoryCLIUsesStrictExitStatus(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "rejected"}[reject], func(t *testing.T) {
			dir := t.TempDir()
			src := `Card(Name: "Land", Typeline: (Supertypes: [], Cardtypes: [Land], Subtypes: [Forest]))`
			if reject {
				src += ` Card(Name: "Gap", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [UnknownRule(DeeplyNested)])`
			}
			if err := os.WriteFile(filepath.Join(dir, "cards.ron"), []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			status := run([]string{"-dir", dir, "-inventory", "-json"}, &out, &stderr)
			expected := 0
			if reject {
				expected = 1
			}
			if status != expected {
				t.Fatalf("status %d: %s", status, &stderr)
			}
			var report ron.Inventory
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.SchemaVersion != 1 || report.Compiled != 1 || report.EngineValidated != nil || report.BehaviorVerified != nil {
				t.Fatalf("report: %+v", report)
			}
			if reject && len(report.Issues) != 1 {
				t.Fatalf("missing rejection: %+v", report)
			}
			out.Reset()
			stderr.Reset()
			if status := run([]string{"-dir", dir, "-json"}, &out, &stderr); status != expected {
				t.Fatalf("strict load status %d", status)
			}
		})
	}
}
