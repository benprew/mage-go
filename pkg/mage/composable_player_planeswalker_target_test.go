package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerOrPlaneswalkerDamageTargetContract(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 4), b.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
	target := DeclarePlayerOrPlaneswalkerTarget("recipient")
	possible := target.spec.Possible(a.PlayerID(), source.Card, g)
	for _, id := range possible {
		if id == creature.ID() {
			t.Fatal("creature was a legal player-or-planeswalker target")
		}
	}
	if !slices.Contains(possible, a.PlayerID()) || !slices.Contains(possible, b.PlayerID()) || !slices.Contains(possible, walker.ID()) {
		t.Fatalf("legal recipients missing: %v", possible)
	}
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(2)), source.ID(), a.PlayerID(), []uuid.UUID{walker.ID()}); err != nil {
		t.Fatal(err)
	}
	if walker.Counters[Loyalty] != 3 || creature.Damage != 0 {
		t.Fatalf("loyalty %d, creature damage %d", walker.Counters[Loyalty], creature.Damage)
	}
	g.PhaseOut(walker.ID())
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(1)), source.ID(), a.PlayerID(), []uuid.UUID{walker.ID()}); err != nil {
		t.Fatal(err)
	}
	if walker.Counters[Loyalty] != 3 {
		t.Fatal("phased planeswalker received damage")
	}
	g.PhaseIn([]uuid.UUID{walker.ID()})
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{walker.ID(): walker.incarnationID}
	g.RemoveFromBattlefield(walker)
	returned := g.PutOnBattlefield(walker.Card, b.PlayerID())
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(1)), source.ID(), a.PlayerID(), []uuid.UUID{returned.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Counters[Loyalty] != 5 {
		t.Fatal("damage followed returned planeswalker")
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, DealDamageFrom(This, target, Fixed(2)), source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone damage or player target failed")
	}
}

func TestPlayerOrPlaneswalkerDamageTargetDeclaration(t *testing.T) {
	target := DeclarePlayerOrPlaneswalkerTarget("recipient")
	valid := NewInstant("Valid", "", Spell(DealDamageFrom(This, target, Fixed(2)), WithTargets(target)))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	missing := NewInstant("Missing", "", Spell(DealDamageFrom(This, target, Fixed(2))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "target binding requires") {
		t.Fatalf("missing target: %v", err)
	}
	binding := DeclarePlayerOrPlaneswalkerTarget("")
	card := NewInstant("Bad", "", Spell(DealDamageFrom(This, binding, Fixed(2)), WithTargets(binding)))
	if err := ValidateCardDefinition("Bad", card); err == nil {
		t.Fatal("accepted an empty target label")
	}
}
