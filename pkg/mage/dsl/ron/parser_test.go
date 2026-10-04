package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCardRecords(t *testing.T) {
	src := `// Two records can span multiple lines.
Card(Name: "Test \\\" name", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: [Human]),
 Rules: [Flying, Activated(PayMana([ManaCostR]), ActionList([]))], CardPT: (Power: +1, Toughness: -0),)
/* outer /* nested */ comment */
Card(Name: r#"Raw \ name"#, Typeline: (Supertypes: [], Cardtypes: [Land], Subtypes: []))`
	nodes, err := Parse(strings.NewReader(src), "cards.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].Name != "Card" || nodes[0].Fields["CardPT"].Fields["Power"].Integer != 1 || nodes[1].Fields["Name"].String != `Raw \ name` {
		t.Fatalf("wrong records: %#v", nodes)
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	for _, src := range []string{
		`Card(Name: "A", Name: "B")`, `Card(Name: "unterminated)`,
		`Card(Name: "A" Rules: [])`, `Card(Rules: [Flying))`,
		`/* unfinished`, `Card(1, Name: "A")`, `Card(Name: "A") !`,
		`Card(Name: "A", CardPT: (Power: 999999999999999999999999999999, Toughness: 1))`,
	} {
		t.Run(src, func(t *testing.T) {
			_, err := Parse(strings.NewReader(src), "bad.ron")
			if err == nil || !strings.Contains(err.Error(), "bad.ron:") {
				t.Fatalf("expected source location, got %v", err)
			}
		})
	}
}

func TestParseRepositoryCards(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil || len(files) != 12 {
		t.Fatalf("card files: %v, %v", files, err)
	}
	total := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if node.Name != "Card" || node.Fields["Name"].String == "" {
				t.Fatalf("invalid card in %s: %#v", file, node)
			}
		}
		total += len(nodes)
	}
	if total != 2445 {
		t.Fatalf("got %d card records, want 2445", total)
	}
}
