package catalog

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestLoadSet(t *testing.T) {
	c, err := LoadSet(filepath.Join(testdataDir(), "LEA.json"))
	if err != nil {
		t.Fatalf("LoadSet: %v", err)
	}
	if c.CardCount() != 3 {
		t.Fatalf("expected 3 cards, got %d", c.CardCount())
	}
}

func TestLoadAll(t *testing.T) {
	c, err := LoadAll(testdataDir())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	// 3 from LEA + 2 from INV = 5
	if c.CardCount() != 5 {
		t.Fatalf("expected 5 cards, got %d", c.CardCount())
	}
}

func TestCardByName(t *testing.T) {
	c := loadTestCatalog(t)

	// Exact name lookup.
	cards := c.CardByName("Black Lotus")
	if len(cards) != 1 {
		t.Fatalf("expected 1 Black Lotus, got %d", len(cards))
	}
	if cards[0].Rarity != "rare" {
		t.Errorf("expected rare, got %s", cards[0].Rarity)
	}

	// Case insensitive.
	cards = c.CardByName("black lotus")
	if len(cards) != 1 {
		t.Fatalf("expected 1 Black Lotus (case-insensitive), got %d", len(cards))
	}

	// Multiple printings.
	cards = c.CardByName("Lightning Bolt")
	if len(cards) != 2 {
		t.Fatalf("expected 2 Lightning Bolt printings, got %d", len(cards))
	}
}

func TestCardBySetAndNumber(t *testing.T) {
	c := loadTestCatalog(t)

	card, ok := c.CardBySetAndNumber("LEA", "232")
	if !ok {
		t.Fatal("expected to find LEA #232")
	}
	if card.Name != "Black Lotus" {
		t.Errorf("expected Black Lotus, got %s", card.Name)
	}

	_, ok = c.CardBySetAndNumber("LEA", "999")
	if ok {
		t.Error("expected not to find LEA #999")
	}
}

func TestCardBySetAndName(t *testing.T) {
	c := loadTestCatalog(t)

	card, ok := c.CardBySetAndName("LEA", "Shivan Dragon")
	if !ok {
		t.Fatal("expected to find Shivan Dragon in LEA")
	}
	if card.Power != "5" {
		t.Errorf("expected power 5, got %s", card.Power)
	}
}

func TestCardsBySet(t *testing.T) {
	c := loadTestCatalog(t)

	cards := c.CardsBySet("LEA")
	if len(cards) != 3 {
		t.Fatalf("expected 3 LEA cards, got %d", len(cards))
	}

	cards = c.CardsBySet("inv")
	if len(cards) != 2 {
		t.Fatalf("expected 2 INV cards, got %d", len(cards))
	}
}

func TestCardsByArtist(t *testing.T) {
	c := loadTestCatalog(t)

	cards := c.CardsByArtist("Christopher Rush")
	if len(cards) != 3 {
		t.Fatalf("expected 3 cards by Christopher Rush, got %d", len(cards))
	}
}

func TestAllPrintings(t *testing.T) {
	c := loadTestCatalog(t)

	// Lightning Bolt has oracle_id "0cf21fad-ad0b-41e0-b7a3-3e0e3f452cf1" in both sets.
	cards := c.AllPrintings("0cf21fad-ad0b-41e0-b7a3-3e0e3f452cf1")
	if len(cards) != 2 {
		t.Fatalf("expected 2 printings, got %d", len(cards))
	}
}

func TestSetInfo(t *testing.T) {
	c := loadTestCatalog(t)

	si, ok := c.GetSetInfo("LEA")
	if !ok {
		t.Fatal("expected to find LEA set info")
	}
	if si.Name != "Limited Edition Alpha" {
		t.Errorf("expected Limited Edition Alpha, got %s", si.Name)
	}
	if si.CardCount != 295 {
		t.Errorf("expected 295 cards, got %d", si.CardCount)
	}

	si, ok = c.GetSetInfo("inv")
	if !ok {
		t.Fatal("expected to find INV set info")
	}
	if si.Block != "Invasion" {
		t.Errorf("expected Invasion block, got %s", si.Block)
	}
}

func TestAllSets(t *testing.T) {
	c := loadTestCatalog(t)

	sets := c.AllSets()
	if len(sets) != 2 {
		t.Fatalf("expected 2 sets, got %d", len(sets))
	}
}

func TestIsReservedList(t *testing.T) {
	c := loadTestCatalog(t)

	if !c.IsReservedList("Black Lotus") {
		t.Error("expected Black Lotus to be reserved")
	}
	if c.IsReservedList("Lightning Bolt") {
		t.Error("expected Lightning Bolt to not be reserved")
	}
}

func TestSplitCardIndexing(t *testing.T) {
	c := loadTestCatalog(t)

	// Full name.
	cards := c.CardByName("Fire // Ice")
	if len(cards) != 1 {
		t.Fatalf("expected 1 result for 'Fire // Ice', got %d", len(cards))
	}

	// Individual face names.
	cards = c.CardByName("Fire")
	if len(cards) != 1 {
		t.Fatalf("expected 1 result for 'Fire', got %d", len(cards))
	}
	if cards[0].Name != "Fire // Ice" {
		t.Errorf("expected Fire // Ice, got %s", cards[0].Name)
	}

	cards = c.CardByName("Ice")
	if len(cards) != 1 {
		t.Fatalf("expected 1 result for 'Ice', got %d", len(cards))
	}
}

func TestEmptyResults(t *testing.T) {
	c := loadTestCatalog(t)

	if cards := c.CardByName("Nonexistent Card"); cards != nil {
		t.Errorf("expected nil for nonexistent card, got %v", cards)
	}
	if cards := c.CardsBySet("ZZZ"); cards != nil {
		t.Errorf("expected nil for nonexistent set, got %v", cards)
	}
	if cards := c.CardsByArtist("Nobody"); cards != nil {
		t.Errorf("expected nil for nonexistent artist, got %v", cards)
	}
}

func loadTestCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := LoadAll(testdataDir())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	return c
}
