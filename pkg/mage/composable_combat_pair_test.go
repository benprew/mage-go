package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatPairPatternBothDirectionsPerOpponent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	watcher := NewCreature("Watcher", "", 2, 4, WithKeyword(core.CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Wall").WithOpponent(opponent),
		AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	watcher.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, b.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	wall := NewCreature("Wall", "", 2, 2, WithSubTypes("Wall"))
	for _, card := range []Card{first, second, wall} {
		g.PutOnBattlefield(card, a.PlayerID())
		g.combat.AddAttacker(card.ID(), b.PlayerID())
	}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: watcher.ID(), AttackerID: first.ID()}, {BlockerID: watcher.ID(), AttackerID: second.ID()}, {BlockerID: watcher.ID(), AttackerID: wall.ID()}})
	if got := g.FindPermanent(first.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("first attacker counters = %d", got)
	}
	if got := g.FindPermanent(second.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("second attacker counters = %d", got)
	}
	if got := g.FindPermanent(wall.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("Wall counters = %d", got)
	}

	g2 := NewGame(NewBasePlayer("C"), NewBasePlayer("D"))
	c, d := g2.players[0], g2.players[1]
	opponent2 := BindPermanent("blocker")
	attacker := NewCreature("Attacker", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Wall").WithOpponent(opponent2),
		AddCountersTo(From(opponent2), core.P1P1, Fixed(1)))))
	g2.PutOnBattlefield(attacker, c.PlayerID())
	g2.combat.AddAttacker(attacker.ID(), d.PlayerID())
	blocker1 := NewCreature("Blocker 1", "", 2, 2)
	blocker2 := NewCreature("Blocker 2", "", 2, 2)
	wall2 := NewCreature("Wall", "", 2, 2, WithSubTypes("Wall"))
	for _, card := range []Card{blocker1, blocker2, wall2} {
		g2.PutOnBattlefield(card, d.PlayerID())
	}
	g2.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker1.ID(), AttackerID: attacker.ID()}, {BlockerID: blocker2.ID(), AttackerID: attacker.ID()}, {BlockerID: wall2.ID(), AttackerID: attacker.ID()}})
	for _, card := range []Card{blocker1, blocker2} {
		if got := g2.FindPermanent(card.ID()).Counters[core.P1P1]; got != 1 {
			t.Fatalf("%s counters = %d", card.Name(), got)
		}
	}
	if got := g2.FindPermanent(wall2.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("Wall blocker counters = %d", got)
	}
}

func TestCombatPairPatternDeclarationValidation(t *testing.T) {
	opponent := BindPermanent("opponent")
	for _, pattern := range []EventPattern{
		BlocksOrBecomesBlockedByCreature(This).OfSubtype(""),
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Island"),
		BlocksOrBecomesBlockedByCreature(This).OfSubtype("Wall").WithoutSubtype("Orc"),
		BlocksOrBecomesBlockedByCreature(This).WithOpponent(nil),
		BlocksOrBecomesBlockedByCreature(This).WithOpponent(opponent).WithOpponent(opponent),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid combat-pair pattern accepted")
		}
	}
	card := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(BlocksOrBecomesBlockedByCreature(This), AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound opponent accepted")
	}
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(BlocksOrBecomesBlockedByCreature(This).WithOpponent(opponent), AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestCombatPairPatternCaptureIdentityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	watcher := NewCreature("Watcher", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithOpponent(opponent),
		AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: watcher.ID(), TargetID: blocker.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(blocker.ID()).Counters[core.P1P1] != 0 || clone.FindPermanent(blocker.ID()).Counters[core.P1P1] != 1 {
		t.Fatal("clone changed original pending capture")
	}
	g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(blocker.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("capture followed a returned opponent")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: watcher.ID(), TargetID: blocker.ID(), PlayerID: a.PlayerID()})
	g.PhaseOut(blocker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanentIncludingPhased(blocker.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("capture affected a phased-out opponent")
	}
}

func TestCombatPairPatternEntryAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	entrant := NewCreature("Entrant", "", 2, 2)
	g.PutOnBattlefieldBlocking(entrant, b.PlayerID(), watcher.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("entry pair life = %d, %d", a.Life(), b.Life())
	}
	g.PhaseOut(watcher.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: watcher.ID(), TargetID: entrant.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("phased-out source observed a pair")
	}
}

func TestCombatPairPatternEventTimeFilterAndDepartedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	watcher := NewCreature("Watcher", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Wall").WithOpponent(opponent),
		AddCountersTo(From(opponent), core.P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: watcher.ID(), TargetID: blocker.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	blocker.AddSubType("Wall")
	g.ResolveStack()
	if got := g.FindPermanent(blocker.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("queued trigger lost event-time match: %d", got)
	}
}
