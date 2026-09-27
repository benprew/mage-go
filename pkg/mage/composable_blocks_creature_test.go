package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlocksCreaturePerPairAndCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("attacker")
	blocker := NewCreature("Blocker", "", 2, 4, WithKeyword(core.CanBlockAny), WithAbility(WhenEvent(
		BlocksCreature(This).WithoutSubtype("Wall").WithOpponent(opponent),
		AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	blocker.SetOwner(b.PlayerID())
	if err := ValidateCardDefinition(blocker.Name(), blocker); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(blocker, b.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	wall := NewCreature("Wall", "", 2, 2, WithSubTypes("Wall"))
	for _, card := range []Card{first, second, wall} {
		g.PutOnBattlefield(card, a.PlayerID())
		g.combat.AddAttacker(card.ID(), b.PlayerID())
	}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}, {BlockerID: blocker.ID(), AttackerID: second.ID()}, {BlockerID: blocker.ID(), AttackerID: wall.ID()}})
	if got := g.FindPermanent(first.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("first attacker counters = %d", got)
	}
	if got := g.FindPermanent(second.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("second attacker counters = %d", got)
	}
	if got := g.FindPermanent(wall.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("Wall counters = %d", got)
	}
}

func TestBlocksCreatureDeclarationValidation(t *testing.T) {
	opponent := BindPermanent("opponent")
	for _, pattern := range []EventPattern{
		BlocksCreature(This).WithoutSubtype(""),
		BlocksCreature(This).WithoutSubtype("Island"),
		BlocksCreature(This).OfSubtype("Wall").WithoutSubtype("Orc"),
		BlocksCreature(This).WithOpponent(nil),
		BlocksCreature(This).WithOpponent(opponent).WithOpponent(opponent),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid event pattern accepted")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(BlocksCreature(This), AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound opponent accepted")
	}
}

func TestBlocksCreatureOpponentIncarnationAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	blocker := NewCreature("Blocker", "", 2, 4, WithAbility(WhenEvent(
		BlocksCreature(This).WithOpponent(opponent),
		AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: blocker.ID(), TargetID: attacker.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(attacker.ID()).Counters[core.P1P1] != 0 || clone.FindPermanent(attacker.ID()).Counters[core.P1P1] != 1 {
		t.Fatal("clone changed the original pending trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(attacker.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("pending trigger followed a returned attacker")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: blocker.ID(), TargetID: attacker.ID(), PlayerID: b.PlayerID()})
	g.PhaseOut(attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanentIncludingPhased(attacker.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("pending trigger affected a phased-out attacker")
	}
}

func TestBlocksCreatureSourcePhasingAtDetection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := NewCreature("Blocker", "", 2, 4, WithAbility(WhenEvent(
		BlocksCreature(This), GainLifeFor(You, Fixed(1)))))
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PhaseOut(blocker.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: blocker.ID(), TargetID: attacker.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("phased-out source observed the block event")
	}
}
