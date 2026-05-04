package mage

import (
	"slices"
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// chooserPlayer wraps BasePlayer to script ChooseCardFromLibrary returns
// (used for hand picks too — both call paths share that method).
type chooserPlayer struct {
	*BasePlayer
	pick      string // name of the card to pick; "" => decline
	calls     int
	lastCands []Card
}

func newChooserPlayer(name string) *chooserPlayer {
	return &chooserPlayer{BasePlayer: NewBasePlayer(name)}
}

func (cp *chooserPlayer) ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card {
	cp.calls++
	cp.lastCands = append([]Card(nil), candidates...)
	if cp.pick == "" {
		return nil
	}
	for _, c := range candidates {
		if c.Name() == cp.pick {
			return c
		}
	}
	return nil
}

func revealTestGame() (*Game, *chooserPlayer, *chooserPlayer) {
	a := newChooserPlayer("A")
	b := newChooserPlayer("B")
	g := NewGame(a, b)
	return g, a, b
}

func TestRevealTopN_ReturnsCopyAndDoesNotMutateLibrary(t *testing.T) {
	g, a, _ := revealTestGame()
	for _, name := range []string{"Forest", "Plains", "Mountain", "Island"} {
		a.AddToLibrary(NewLand(name))
	}
	pre := len(a.Library())
	revealed := g.RevealTopN(a, 3)
	if len(revealed) != 3 {
		t.Fatalf("expected 3 revealed, got %d", len(revealed))
	}
	if got := []string{revealed[0].Name(), revealed[1].Name(), revealed[2].Name()}; got[0] != "Forest" || got[1] != "Plains" || got[2] != "Mountain" {
		t.Errorf("unexpected order: %v", got)
	}
	if len(a.Library()) != pre {
		t.Errorf("library mutated: pre=%d post=%d", pre, len(a.Library()))
	}
	revealed[0] = nil
	if a.Library()[0] == nil {
		t.Error("returned slice aliases library; expected a copy")
	}
}

func TestRevealTopN_ClampsToLibrarySize(t *testing.T) {
	g, a, _ := revealTestGame()
	a.AddToLibrary(NewLand("Forest"))
	a.AddToLibrary(NewLand("Plains"))
	got := g.RevealTopN(a, 10)
	if len(got) != 2 {
		t.Errorf("expected 2 (clamped), got %d", len(got))
	}
}

func TestRevealTopN_EmptyAndZero(t *testing.T) {
	g, a, _ := revealTestGame()
	if got := g.RevealTopN(a, 5); got != nil {
		t.Errorf("empty library: got %v want nil", got)
	}
	a.AddToLibrary(NewLand("Forest"))
	if got := g.RevealTopN(a, 0); got != nil {
		t.Errorf("n=0: got %v want nil", got)
	}
	if got := g.RevealTopN(nil, 1); got != nil {
		t.Errorf("nil player: got %v want nil", got)
	}
}

func TestRemoveTopN_RemovesFromLibrary(t *testing.T) {
	g, a, _ := revealTestGame()
	a.AddToLibrary(NewLand("Forest"))
	a.AddToLibrary(NewLand("Plains"))
	a.AddToLibrary(NewLand("Mountain"))
	taken := g.RemoveTopN(a, 2)
	if len(taken) != 2 || taken[0].Name() != "Forest" || taken[1].Name() != "Plains" {
		t.Errorf("unexpected taken: %v", taken)
	}
	if len(a.Library()) != 1 || a.Library()[0].Name() != "Mountain" {
		t.Errorf("unexpected remaining library: %v", a.Library())
	}
}

func TestRevealAndPickFromTop_ChoosesMatchingCard(t *testing.T) {
	g, a, _ := revealTestGame()
	a.AddToLibrary(NewLand("Forest"))
	creature := NewCreature("Goblin", "{R}", 1, 1)
	a.AddToLibrary(creature)
	a.AddToLibrary(NewLand("Plains"))
	creatureFilter := IsCreatureCard
	a.pick = "Goblin"
	chosen, revealed := g.RevealAndPickFromTop(a, a, 3, creatureFilter, false, "test pick")
	if chosen == nil || chosen.Name() != "Goblin" {
		t.Fatalf("expected Goblin chosen, got %v", chosen)
	}
	if len(revealed) != 3 {
		t.Errorf("expected 3 revealed, got %d", len(revealed))
	}
	if a.calls != 1 {
		t.Errorf("expected 1 call, got %d", a.calls)
	}
	// Only the creature should have been offered.
	if len(a.lastCands) != 1 || a.lastCands[0].Name() != "Goblin" {
		t.Errorf("expected only creature offered, got %v", a.lastCands)
	}
}

func TestRevealAndPickFromTop_NoMatchReturnsNil(t *testing.T) {
	g, a, _ := revealTestGame()
	a.AddToLibrary(NewLand("Forest"))
	a.AddToLibrary(NewLand("Plains"))
	creatureFilter := IsCreatureCard
	chosen, revealed := g.RevealAndPickFromTop(a, a, 2, creatureFilter, false, "test")
	if chosen != nil {
		t.Errorf("expected nil chosen, got %v", chosen)
	}
	if len(revealed) != 2 {
		t.Errorf("expected 2 revealed, got %d", len(revealed))
	}
	if a.calls != 0 {
		t.Error("chooser must not be invoked when no candidates match")
	}
}

func TestRevealAndPickFromTop_MayDeclineHonored(t *testing.T) {
	g, a, _ := revealTestGame()
	creature := NewCreature("Goblin", "{R}", 1, 1)
	a.AddToLibrary(creature)
	cf := IsCreatureCard
	a.pick = "" // decline
	chosen, _ := g.RevealAndPickFromTop(a, a, 1, cf, true, "test")
	if chosen != nil {
		t.Errorf("expected decline to return nil, got %v", chosen)
	}
}

func TestRevealAndPickFromTop_MayDeclineFalseForcesPick(t *testing.T) {
	g, a, _ := revealTestGame()
	creature := NewCreature("Goblin", "{R}", 1, 1)
	a.AddToLibrary(creature)
	cf := IsCreatureCard
	a.pick = "" // try to decline
	chosen, _ := g.RevealAndPickFromTop(a, a, 1, cf, false, "test")
	if chosen == nil {
		t.Errorf("mayDecline=false: expected fallback pick, got nil")
	}
}

func TestPutOnBottomInRandomOrder_PlacesAllAtBottom(t *testing.T) {
	g, a, _ := revealTestGame()
	a.AddToLibrary(NewLand("Forest"))
	a.AddToLibrary(NewLand("Plains"))
	cards := []Card{NewLand("Mountain"), NewLand("Island"), NewLand("Swamp")}
	g.PutOnBottomInRandomOrder(a, cards)
	lib := a.Library()
	if len(lib) != 5 {
		t.Fatalf("expected 5 cards, got %d", len(lib))
	}
	if lib[0].Name() != "Forest" || lib[1].Name() != "Plains" {
		t.Errorf("top two unchanged check failed: %v %v", lib[0].Name(), lib[1].Name())
	}
	bottom := map[string]bool{lib[2].Name(): true, lib[3].Name(): true, lib[4].Name(): true}
	for _, want := range []string{"Mountain", "Island", "Swamp"} {
		if !bottom[want] {
			t.Errorf("missing %s at bottom; got %v", want, bottom)
		}
	}
}

func TestPutOnTopInChosenOrder_PlacesInOrder(t *testing.T) {
	g, a, _ := revealTestGame()
	a.AddToLibrary(NewLand("Forest"))
	cards := []Card{NewLand("Plains"), NewLand("Mountain")}
	g.PutOnTopInChosenOrder(a, cards)
	lib := a.Library()
	if len(lib) != 3 || lib[0].Name() != "Plains" || lib[1].Name() != "Mountain" || lib[2].Name() != "Forest" {
		t.Errorf("unexpected order: %v %v %v", lib[0].Name(), lib[1].Name(), lib[2].Name())
	}
}

func TestRevealHand_ReturnsCopy(t *testing.T) {
	g, a, b := revealTestGame()
	b.AddToHand(NewLand("Forest"))
	b.AddToHand(NewCreature("Goblin", "{R}", 1, 1))
	revealed := g.RevealHand(a, b)
	if len(revealed) != 2 {
		t.Fatalf("expected 2, got %d", len(revealed))
	}
	revealed[0] = nil
	if b.Hand()[0] == nil {
		t.Error("RevealHand returned alias instead of copy")
	}
}

func TestPickFromHand_ChoosesMatchingCard(t *testing.T) {
	g, a, b := revealTestGame()
	b.AddToHand(NewLand("Forest"))
	target := NewCreature("Goblin", "{R}", 1, 1)
	b.AddToHand(target)
	b.AddToHand(NewLand("Plains"))
	noncreature := NewCardFilter("noncreature", func(c Card) bool {
		return !slices.Contains(c.Types(), TypeCreature)
	})
	a.pick = "Forest"
	chosen := g.PickFromHand(a, b, noncreature, false, "you choose a noncreature")
	if chosen == nil || chosen.Name() != "Forest" {
		t.Errorf("expected Forest, got %v", chosen)
	}
	// The creature must NOT have been offered as a candidate.
	for _, c := range a.lastCands {
		if c.Name() == "Goblin" {
			t.Error("filter not applied; creature offered as candidate")
		}
	}
	// Hand unchanged (caller must move the card).
	if len(b.Hand()) != 3 {
		t.Errorf("hand size changed: %d", len(b.Hand()))
	}
}

func TestPickFromHand_NoMatchReturnsNil(t *testing.T) {
	g, a, b := revealTestGame()
	b.AddToHand(NewCreature("Goblin", "{R}", 1, 1))
	noncreature := NewCardFilter("noncreature", func(c Card) bool {
		return !slices.Contains(c.Types(), TypeCreature)
	})
	chosen := g.PickFromHand(a, b, noncreature, false, "test")
	if chosen != nil {
		t.Errorf("expected nil, got %v", chosen)
	}
	if a.calls != 0 {
		t.Error("chooser must not be invoked when no candidates match")
	}
}
