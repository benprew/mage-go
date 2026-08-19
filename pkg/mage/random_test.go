package mage

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func randomTestGame() (*Game, *BasePlayer, *BasePlayer) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	return g, a, b
}

func TestRandIntn_ScriptedResultsAndSafeBounds(t *testing.T) {
	g, _, _ := randomTestGame()
	results := []int{5, -1, 8, 7}
	g.SetRandomResults(results)
	results[0] = 0

	if got := g.RandIntn(3); got != 2 {
		t.Errorf("RandIntn(3) = %d, want 2", got)
	}
	if got := g.RandIntn(4); got != 3 {
		t.Errorf("RandIntn(4) with scripted -1 = %d, want 3", got)
	}
	if got := g.RandIntn(5); got != 3 {
		t.Errorf("RandIntn(5) = %d, want 3", got)
	}
	if got := g.RandIntn(0); got != 0 {
		t.Errorf("RandIntn(0) = %d, want 0", got)
	}
	if got := g.RandIntn(-2); got != 0 {
		t.Errorf("RandIntn(-2) = %d, want 0", got)
	}
	if got := g.RandIntn(4); got != 3 {
		t.Errorf("nonpositive bound consumed scripted result: got %d, want 3", got)
	}
}

func TestRandIntn_CloneHasIndependentScriptedResults(t *testing.T) {
	g, _, _ := randomTestGame()
	g.SetRandomResults([]int{4, 5})
	clone := g.Clone()

	if got := g.RandIntn(3); got != 1 {
		t.Fatalf("original first result = %d, want 1", got)
	}
	if got := clone.RandIntn(3); got != 1 {
		t.Fatalf("clone first result = %d, want 1", got)
	}
	if got := g.RandIntn(3); got != 2 {
		t.Errorf("original second result = %d, want 2", got)
	}
	if got := clone.RandIntn(3); got != 2 {
		t.Errorf("clone second result = %d, want 2", got)
	}
}

func TestRandomPlayerUsesGameScriptedResult(t *testing.T) {
	g, a, b := randomTestGame()
	g.SetRandomResults([]int{1, 0})

	if got := g.RandomPlayer(); got == nil || got.PlayerID() != b.PlayerID() {
		t.Fatalf("first random player = %v, want B", got)
	}
	if got := g.RandomPlayer(); got == nil || got.PlayerID() != a.PlayerID() {
		t.Fatalf("second random player = %v, want A", got)
	}
}

func TestRandomPlayerReturnsNilWithoutPlayers(t *testing.T) {
	if got := (&Game{}).RandomPlayer(); got != nil {
		t.Fatalf("random player in empty game = %v, want nil", got)
	}
}

func TestRandomPermanentUsesFilterAndScriptedResult(t *testing.T) {
	g, a, _ := randomTestGame()
	first := addRandomTargetCreature(g, a, "First Creature")
	artifact := NewArtifact("Artifact", "{1}")
	artifact.SetOwner(a.PlayerID())
	g.PutOnBattlefield(artifact, a.PlayerID())
	second := addRandomTargetCreature(g, a, "Second Creature")
	g.SetRandomResults([]int{1})

	if got := g.RandomPermanent(IsCreature); got == nil || got.ID() != second.ID() {
		t.Fatalf("random creature = %v, want %v (first was %v)", got, second.ID(), first.ID())
	}
}

func TestRandomPermanentReturnsNilWithoutMatchAndDoesNotConsumeRandomness(t *testing.T) {
	g, a, b := randomTestGame()
	g.SetRandomResults([]int{1})

	if got := g.RandomPermanent(IsCreature); got != nil {
		t.Fatalf("random creature = %v, want nil", got)
	}
	if got := g.RandomPlayer(); got == nil || got.PlayerID() != b.PlayerID() {
		t.Fatalf("random result was consumed by empty permanent selection: got %v, want B (A was %v)", got, a.PlayerID())
	}
}

func TestRandomDamageTargetChoosesUniformlyFromCreaturesAndPlayers(t *testing.T) {
	g, a, b := randomTestGame()
	first := addRandomTargetCreature(g, a, "First Damage Target")
	second := addRandomTargetCreature(g, b, "Second Damage Target")
	g.SetRandomResults([]int{0, 1, 2, 3})

	want := []uuid.UUID{first.ID(), second.ID(), a.PlayerID(), b.PlayerID()}
	for i, expected := range want {
		if got := g.RandomDamageTarget(); got != expected {
			t.Fatalf("random damage target %d = %v, want %v", i, got, expected)
		}
	}
}

func TestRandomDamageTargetReturnsNilWithoutCandidates(t *testing.T) {
	if got := (&Game{}).RandomDamageTarget(); got != uuid.Nil {
		t.Fatalf("random damage target in empty game = %v, want nil UUID", got)
	}
}

func TestRandomColorUsesGameScriptedResult(t *testing.T) {
	g, _, _ := randomTestGame()
	g.SetRandomResults([]int{0, 1, 2, 3, 4, -1})

	want := []Color{White, Blue, Black, Red, Green, Green}
	for i, expected := range want {
		if got := g.RandomColor(); got != expected {
			t.Fatalf("random color %d = %v, want %v", i, got, expected)
		}
	}
}

func TestRandomSpellOrPermanentIncludesBothZones(t *testing.T) {
	g, a, _ := randomTestGame()
	permanent := addRandomTargetCreature(g, a, "Permanent")
	spell := NewInstant("Spell", "{0}", NewSpellAbility())
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatalf("cast spell: %v", err)
	}
	g.SetRandomResults([]int{0, 1})

	if got := g.RandomSpellOrPermanent(); got != permanent.ID() {
		t.Fatalf("first random object = %v, want permanent %v", got, permanent.ID())
	}
	if got := g.RandomSpellOrPermanent(); got != spell.ID() {
		t.Fatalf("second random object = %v, want spell %v", got, spell.ID())
	}
}

func TestRandomHelpersUseGameScriptedResults(t *testing.T) {
	g, a, _ := randomTestGame()
	forest := NewLand("Forest")
	plains := NewLand("Plains")
	mountain := NewLand("Mountain")
	a.AddToHand(forest)
	a.AddToHand(plains)
	a.AddToHand(mountain)
	g.SetRandomResults([]int{1, 1})

	discarded := g.DiscardAtRandom(a, 1)
	if len(discarded) != 1 || discarded[0].Name() != "Plains" {
		t.Fatalf("scripted discard = %v, want Plains", discarded)
	}
	if got := g.RandomCardFromHand(a, CardFilter{}); got == nil || got.Name() != "Mountain" {
		t.Errorf("scripted hand choice = %v, want Mountain", got)
	}
}

func TestFlipCoinUsesGameScriptedRandomResult(t *testing.T) {
	g, a, _ := randomTestGame()
	g.SetRandomResults([]int{0, 1})

	if !g.FlipCoin(a.PlayerID()) {
		t.Error("scripted 0 should be heads")
	}
	if g.FlipCoin(a.PlayerID()) {
		t.Error("scripted 1 should be tails")
	}
}

func TestDiscardAtRandom_DiscardsRequestedCount(t *testing.T) {
	rand.Seed(1) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
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
	rand.Seed(2) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
	g, a, _ := randomTestGame()
	a.AddToGraveyard(NewLand("Forest"))
	zombie := NewCreature("Zombie A", "{1}{B}", 2, 2, WithSubTypes("Zombie"))
	a.AddToGraveyard(zombie)
	a.AddToGraveyard(NewLand("Swamp"))
	zombieFilter := NewCardFilter("zombie", func(c Card) bool {
		if !c.HasType(TypeCreature) {
			return false
		}
		return slices.Contains(c.SubTypes(), "Zombie")
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
		return slices.Contains(c.SubTypes(), "Zombie")
	})
	if got := g.RandomCardFromGraveyard(a, zombieFilter); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestRandomCardFromGraveyard_AnyCardWithZeroFilter(t *testing.T) {
	rand.Seed(3) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
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
	rand.Seed(42) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
	g, a, _ := randomTestGame()
	a.AddToGraveyard(NewLand("Forest"))
	a.AddToGraveyard(NewLand("Plains"))
	a.AddToGraveyard(NewLand("Mountain"))
	counts := map[string]int{}
	for range 3000 {
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
	rand.Seed(4) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
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
