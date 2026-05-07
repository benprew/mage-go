package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRogueDeck(t *testing.T) {
	content := `name = "Test Rogue"
level = 5
main_cards = [
["4",	"Lightning Bolt"],
["20",	"Mountain"],
]
sideboard_cards = [
["2",	"Shatter"],
]
walking_sprite = "foo.png"
face = "bar.png"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test_rogue.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deck, err := ParseRogueDeck(path)
	if err != nil {
		t.Fatal(err)
	}

	if deck.Name != "Test Rogue" {
		t.Errorf("name = %q, want %q", deck.Name, "Test Rogue")
	}
	if deck.Level != 5 {
		t.Errorf("level = %d, want 5", deck.Level)
	}
	if len(deck.MainCards) != 2 {
		t.Fatalf("main_cards len = %d, want 2", len(deck.MainCards))
	}
	if deck.MainCards[0].Name != "Lightning Bolt" || deck.MainCards[0].Count != 4 {
		t.Errorf("main_cards[0] = %+v, want {Lightning Bolt, 4}", deck.MainCards[0])
	}
	if deck.MainCards[1].Name != "Mountain" || deck.MainCards[1].Count != 20 {
		t.Errorf("main_cards[1] = %+v, want {Mountain, 20}", deck.MainCards[1])
	}
	if len(deck.Sideboard) != 1 {
		t.Fatalf("sideboard len = %d, want 1", len(deck.Sideboard))
	}
	if deck.Sideboard[0].Name != "Shatter" || deck.Sideboard[0].Count != 2 {
		t.Errorf("sideboard[0] = %+v, want {Shatter, 2}", deck.Sideboard[0])
	}
	if deck.SourceFile != "test_rogue.toml" {
		t.Errorf("source_file = %q, want %q", deck.SourceFile, "test_rogue.toml")
	}
}

func TestParseRogueDeckEmptySideboard(t *testing.T) {
	content := `name = "Arzakon"
level = 12
main_cards = [
["6",	"Mountain"],
["2",	"Brass Man"],
]
sideboard_cards = [
]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "arzakon.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deck, err := ParseRogueDeck(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.MainCards) != 2 {
		t.Fatalf("main_cards len = %d, want 2", len(deck.MainCards))
	}
	if len(deck.Sideboard) != 0 {
		t.Errorf("sideboard len = %d, want 0", len(deck.Sideboard))
	}
}

func TestParseCardEntryUnicode(t *testing.T) {
	entry, err := parseCardEntry(`["2",   "El-Hajjâj"],`)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Name != "El-Hajjâj" || entry.Count != 2 {
		t.Errorf("got %+v, want {El-Hajjâj, 2}", entry)
	}
}

func TestLoadAllRogueDecks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.toml", "b.toml"} {
		content := `name = "` + name + `"
main_cards = [
["10", "Plains"],
]
sideboard_cards = [
]
`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Non-toml file should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	decks, err := LoadAllRogueDecks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(decks) != 2 {
		t.Errorf("loaded %d decks, want 2", len(decks))
	}
}
