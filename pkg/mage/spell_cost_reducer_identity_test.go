package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCostReducerSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Reducer", "", WithStaticAbility(ReduceSpellCostStatic(SpellAny(), FixedAmount(2), nil)))
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	probe := NewInstant("Probe", "{3}{U}")
	check := func(game *Game, player uuid.UUID, want int) {
		t.Helper()
		if got := game.ConditionalSpellCostReduction(player, probe); got != want {
			t.Fatalf("reduction = %d, want %d", got, want)
		}
	}
	check(g, a.PlayerID(), 2)
	saved := g.effects.Rules.SpellCostReducers[0]
	clone := g.Clone()
	g.PhaseOut(source.ID())
	check(g, a.PlayerID(), 0)
	check(clone, a.PlayerID(), 2)
	g.PhaseIn([]uuid.UUID{source.ID()})
	check(g, a.PlayerID(), 2)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if saved.applies(g, a.PlayerID(), probe) {
		t.Fatal("departed source still applies")
	}
	g.PutOnBattlefield(card, b.PlayerID())
	if saved.applies(g, a.PlayerID(), probe) || saved.applies(g, b.PlayerID(), probe) {
		t.Fatal("returned source revived old reducer")
	}
	check(g, a.PlayerID(), 0)
	check(g, b.PlayerID(), 2)
	check(clone, a.PlayerID(), 2)
}

func TestSpellCostReducerPhasedCastingRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(NewArtifact("Reducer", "", WithStaticAbility(ReduceSpellCostStatic(SpellAny(), FixedAmount(2), nil))), a.PlayerID())
	spell := NewInstant("Probe", "{2}{U}", Spell(GainLifeFor(You, Fixed(1))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	g.PhaseOut(source.ID())
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("phased reducer paid for spell")
	}
	if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment changed resources")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
		t.Fatal("phase-in did not restore payment")
	}
}
