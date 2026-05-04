package mage

import (
	"math/rand"
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func randomTestGame() (*Game, *BasePlayer, *BasePlayer) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	return g, a, b
}

func TestDiscardAtRandom_DiscardsRequestedCount(t *testing.T) {
	rand.Seed(1)
	g, a, _ := randomTestGame()
	for _, n := range []string{"Forest", "Plains", "Mountain", "Island", "Swamp"} {
		a.AddToHand(NewLand(n))
	}
	discarded := g.DiscardAtRandom(a, 3)
	if len(discarded) != 3 {
		t.Fatalf("expected 3 discards, got %d", len(discarded))
	}
	if len(a.Hand()) != 2 {
		t.Errorf("expected 2 cards left in hand, got %d", len(a.Hand()))
	}
	if len(a.Graveyard()) != 3 {
		t.Errorf("expected 3 cards in graveyard, got %d", len(a.Graveyard()))
	}
}

func TestDiscardAtRandom_StopsAtEmptyHand(t *testing.T) {
	g, a, _ := randomTestGame()
	a.AddToHand(NewLand("Forest"))
	a.AddToHand(NewLand("Plains"))
	got := g.DiscardAtRandom(a, 5)
	if len(got) != 2 {
		t.Errorf("expected 2 (clamped to hand), got %d", len(got))
	}
	if len(a.Hand()) != 0 {
		t.Errorf("expected empty hand, got %d", len(a.Hand()))
	}
}

func TestDiscardAtRandom_NilAndZero(t *testing.T) {
	g, a, _ := randomTestGame()
	a.AddToHand(NewLand("Forest"))
	if got := g.DiscardAtRandom(nil, 1); got != nil {
		t.Errorf("nil player: got %v want nil", got)
	}
	if got := g.DiscardAtRandom(a, 0); got != nil {
		t.Errorf("n=0: got %v want nil", got)
	}
	if len(a.Hand()) != 1 {
		t.Errorf("hand mutated unexpectedly: %d", len(a.Hand()))
	}
}

func TestRandomCardFromGraveyard_FiltersAndDoesNotRemove(t *testing.T) {
	rand.Seed(2)
	g, a, _ := randomTestGame()
	a.AddToGraveyard(NewLand("Forest"))
	zombie := NewCreature("Zombie A", "{1}{B}", 2, 2, WithSubTypes("Zombie"))
	a.AddToGraveyard(zombie)
	a.AddToGraveyard(NewLand("Swamp"))
	zombieFilter := NewCardFilter("zombie", func(c Card) bool {
		if !c.HasType(TypeCreature) {
			return false
		}
		for _, st := range c.SubTypes() {
			if st == "Zombie" {
				return true
			}
		}
		return false
	})
	chosen := g.RandomCardFromGraveyard(a, zombieFilter)
	if chosen == nil || chosen.Name() != "Zombie A" {
		t.Errorf("expected Zombie A, got %v", chosen)
	}
	// graveyard not mutated
	if len(a.Graveyard()) != 3 {
		t.Errorf("graveyard mutated: %d", len(a.Graveyard()))
	}
}

func TestRandomCardFromGraveyard_NoMatchReturnsNil(t *testing.T) {
	g, a, _ := randomTestGame()
	a.AddToGraveyard(NewLand("Forest"))
	zombieFilter := NewCardFilter("zombie", func(c Card) bool {
		if !c.HasType(TypeCreature) {
			return false
		}
		for _, st := range c.SubTypes() {
			if st == "Zombie" {
				return true
			}
		}
		return false
	})
	if got := g.RandomCardFromGraveyard(a, zombieFilter); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestRandomCardFromGraveyard_AnyCardWithZeroFilter(t *testing.T) {
	rand.Seed(3)
	g, a, _ := randomTestGame()
	a.AddToGraveyard(NewLand("Forest"))
	a.AddToGraveyard(NewLand("Plains"))
	a.AddToGraveyard(NewLand("Mountain"))
	got := g.RandomCardFromGraveyard(a, CardFilter{})
	if got == nil {
		t.Fatal("expected a card, got nil")
	}
	// uniform: across many runs we expect each to appear; one sample test
	// just verifies non-nil and that returned card is in the graveyard.
	found := false
	for _, c := range a.Graveyard() {
		if c.ID() == got.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Error("returned card not in graveyard")
	}
}

func TestRandomCardFromGraveyard_UniformDistribution(t *testing.T) {
	rand.Seed(42)
	g, a, _ := randomTestGame()
	a.AddToGraveyard(NewLand("Forest"))
	a.AddToGraveyard(NewLand("Plains"))
	a.AddToGraveyard(NewLand("Mountain"))
	counts := map[string]int{}
	for i := 0; i < 3000; i++ {
		c := g.RandomCardFromGraveyard(a, CardFilter{})
		counts[c.Name()]++
	}
	for name, n := range counts {
		// Expect ~1000 each; allow generous bounds for a 3000-trial sample.
		if n < 800 || n > 1200 {
			t.Errorf("%s appeared %d times, expected near 1000", name, n)
		}
	}
}

func TestRandomCardFromHand_FiltersAndDoesNotRemove(t *testing.T) {
	rand.Seed(4)
	g, a, _ := randomTestGame()
	a.AddToHand(NewLand("Forest"))
	creature := NewCreature("Goblin", "{R}", 1, 1)
	a.AddToHand(creature)
	chosen := g.RandomCardFromHand(a, IsCreatureCard)
	if chosen == nil || chosen.Name() != "Goblin" {
		t.Errorf("expected Goblin, got %v", chosen)
	}
	if len(a.Hand()) != 2 {
		t.Errorf("hand size changed: %d", len(a.Hand()))
	}
}

func TestRandomCardFromHand_NoMatchReturnsNil(t *testing.T) {
	g, a, _ := randomTestGame()
	a.AddToHand(NewLand("Forest"))
	if got := g.RandomCardFromHand(a, IsCreatureCard); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}
