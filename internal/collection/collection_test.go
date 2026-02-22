package collection

import (
	"os"
	"testing"
	"time"

	"github.com/mage/mage/internal/worlddata"
)

func TestLoadNewPlayer(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(dir, "fp_new")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil collection for new player")
	}
	if len(c.Cards) != 0 {
		t.Errorf("expected empty cards, got %v", c.Cards)
	}
	if c.Gold != 0 {
		t.Errorf("expected 0 gold, got %d", c.Gold)
	}
	if len(c.Decks) != 0 {
		t.Errorf("expected no decks, got %d", len(c.Decks))
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	fp := "fp_abc123"

	c := New()
	c.Add([]string{"Lightning Bolt", "Lightning Bolt", "Serra Angel"})
	c.AddGold(42)
	c.Decks = append(c.Decks, SavedDeck{
		Name: "Burn",
		Main: []worlddata.DeckEntry{
			{Name: "Mountain", Count: 20},
			{Name: "Lightning Bolt", Count: 4},
		},
	})
	c.ActiveDeck = 0

	if err := c.Save(dir, fp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir, fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Cards["Lightning Bolt"] != 2 {
		t.Errorf("expected 2x Lightning Bolt, got %d", loaded.Cards["Lightning Bolt"])
	}
	if loaded.Cards["Serra Angel"] != 1 {
		t.Errorf("expected 1x Serra Angel, got %d", loaded.Cards["Serra Angel"])
	}
	if loaded.Gold != 42 {
		t.Errorf("expected 42 gold, got %d", loaded.Gold)
	}
	if len(loaded.Decks) != 1 {
		t.Fatalf("expected 1 deck, got %d", len(loaded.Decks))
	}
	if loaded.Decks[0].Name != "Burn" {
		t.Errorf("expected deck name Burn, got %q", loaded.Decks[0].Name)
	}
	if loaded.ActiveDeck != 0 {
		t.Errorf("expected ActiveDeck 0, got %d", loaded.ActiveDeck)
	}
}

func TestSaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	fp := "fp_nested"

	c := New()
	if err := c.Save(dir, fp); err != nil {
		t.Fatalf("Save should create missing directories: %v", err)
	}

	path := dataPath(dir, fp)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}

func TestSaveOverwritesPreviousSave(t *testing.T) {
	dir := t.TempDir()
	fp := "fp_overwrite"

	c := New()
	c.AddGold(10)
	if err := c.Save(dir, fp); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	c.AddGold(5)
	if err := c.Save(dir, fp); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	loaded, err := Load(dir, fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Gold != 15 {
		t.Errorf("expected 15 gold after two saves, got %d", loaded.Gold)
	}
}

func TestSavePreservesLastBooster(t *testing.T) {
	dir := t.TempDir()
	fp := "fp_booster"

	c := New()
	c.ClaimBooster()
	before := c.LastBooster

	if err := c.Save(dir, fp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir, fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Allow 1-second tolerance for JSON time precision.
	diff := loaded.LastBooster.Sub(before)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("LastBooster not preserved: got %v, want ~%v", loaded.LastBooster, before)
	}
	if loaded.CanClaimBooster() {
		t.Error("should not be able to claim booster immediately after claiming one")
	}
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	fp := "fp_corrupt"

	path := dataPath(dir, fp)
	if err := os.MkdirAll(dataPath(dir, fp)[:len(dataPath(dir, fp))-len("/collection.json")], 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not valid json {{{"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir, fp)
	if err == nil {
		t.Error("expected error loading corrupt file, got nil")
	}
}
