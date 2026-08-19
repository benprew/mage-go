package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestTargetCreatureOfSourceChosenSubtype(t *testing.T) {
	g, a, b := randomTestGame()
	sourceCard := NewCreature("Source", "{1}", 1, 1)
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	source.ChosenSubtype = "Dragon"
	dragon := NewCreature("Dragon", "{1}", 1, 1, WithSubTypes("Dragon"))
	dragon.SetOwner(b.PlayerID())
	dragonPermanent := g.PutOnBattlefield(dragon, b.PlayerID())
	knight := NewCreature("Knight", "{1}", 1, 1, WithSubTypes("Knight"))
	knight.SetOwner(b.PlayerID())
	knightPermanent := g.PutOnBattlefield(knight, b.PlayerID())

	possible := TargetCreatureOfSourceChosenSubtype().Possible(a.PlayerID(), sourceCard, g)
	if !slices.Contains(possible, dragonPermanent.ID()) || slices.Contains(possible, knightPermanent.ID()) {
		t.Fatalf("possible targets = %v, want only matching subtype %v", possible, dragonPermanent.ID())
	}
}

func TestChooseRandomCreatureSubtypeFromTargetLibrary(t *testing.T) {
	g, a, b := randomTestGame()
	sourceCard := NewCreature("Source", "{1}", 1, 1)
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	b.AddToLibrary(NewCreature("Knight", "{1}", 1, 1, WithSubTypes("Human", "Knight")))
	b.AddToLibrary(NewCreature("Dragon", "{1}", 1, 1, WithSubTypes("Dragon")))
	g.SetRandomResults([]int{2})

	if err := ApplyEffect(g, ChooseRandomCreatureSubtypeFromTargetLibrary(), source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatalf("choose subtype: %v", err)
	}
	if source.ChosenSubtype != "Dragon" {
		t.Fatalf("chosen subtype = %q, want Dragon", source.ChosenSubtype)
	}
}
