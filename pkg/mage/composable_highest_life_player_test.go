package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestPlayerWithUniqueHighestLifeResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ref := PlayerWithUniqueHighestLife()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if _, ok := ref.resolvePlayer(ctx); ok {
		t.Fatal("tied life totals selected a player")
	}
	if err := ApplyEffect(g, Sequence(LoseLifeFor(You, Fixed(1)), GainLifeFor(ref, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 22 {
		t.Fatalf("resolution-time selection changed life totals to %d, %d", a.Life(), b.Life())
	}
	if err := ApplyEffect(g, LoseLifeFor(ref, Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("highest-life selection did not use the current game state")
	}
	if err := ApplyEffect(g, GainLifeFor(ref, Fixed(4)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("tied selection was not absent")
	}
}

func TestPlayerWithUniqueHighestLifeCloneAndDefinition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ref := PlayerWithUniqueHighestLife()
	if info := InspectDefinition(ref); info.Role != "player-ref" || info.Text == "" {
		t.Fatalf("selector inspection = %+v", info)
	}
	card := NewArtifact("Watcher", "", WithAbility(Activate(Costs(), GainLifeFor(ref, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).GainLife(2)
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone did not select its own highest-life player")
	}
	a.GainLife(2)
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("pending activation did not select at resolution")
	}
}
