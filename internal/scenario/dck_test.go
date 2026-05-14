package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDCKDeck(t *testing.T) {
	content := `NAME:War Mage
23 [2ED:297] Mountain
4 [2ED:162] Lightning Bolt
2 [4ED:292] Aladdin's Ring
SB: 3 [4ED:184] Detonate
SB: 1 [2ED:170] Red Elemental Blast
`
	dir := t.TempDir()
	path := filepath.Join(dir, "war_mage.dck")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deck, err := ParseDCKDeck(path)
	if err != nil {
		t.Fatal(err)
	}

	if deck.Name != "War Mage" {
		t.Errorf("name = %q, want %q", deck.Name, "War Mage")
	}
	if len(deck.MainCards) != 3 {
		t.Fatalf("main_cards len = %d, want 3", len(deck.MainCards))
	}
	if deck.MainCards[0].Name != "Mountain" || deck.MainCards[0].Count != 23 {
		t.Errorf("main_cards[0] = %+v, want {Mountain, 23}", deck.MainCards[0])
	}
	if deck.MainCards[2].Name != "Aladdin's Ring" || deck.MainCards[2].Count != 2 {
		t.Errorf("main_cards[2] = %+v, want {Aladdin's Ring, 2}", deck.MainCards[2])
	}
	if len(deck.Sideboard) != 2 {
		t.Fatalf("sideboard len = %d, want 2", len(deck.Sideboard))
	}
	if deck.Sideboard[1].Name != "Red Elemental Blast" || deck.Sideboard[1].Count != 1 {
		t.Errorf("sideboard[1] = %+v, want {Red Elemental Blast, 1}", deck.Sideboard[1])
	}
	if deck.SourceFile != "war_mage.dck" {
		t.Errorf("source_file = %q, want %q", deck.SourceFile, "war_mage.dck")
	}
}

func TestLoadAllDCKDecks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.dck", "b.dck"} {
		content := `NAME:` + name + `
10 [2ED:288] Plains
`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.toml"), []byte("name = \"ignore\""), 0644); err != nil {
		t.Fatal(err)
	}

	decks, err := LoadAllDCKDecks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(decks) != 2 {
		t.Errorf("loaded %d decks, want 2", len(decks))
	}
}
