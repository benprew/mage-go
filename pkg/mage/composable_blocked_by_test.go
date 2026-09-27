package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockedByPatternPairsAndBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocking Wall")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(
		BecomesBlockedBy(This).OfSubtype("Wall").WithBlocker(blocker),
		AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall1 := NewCreature("Wall 1", "", 1, 4, WithSubTypes("Wall"))
	wall2 := NewCreature("Wall 2", "", 1, 4, WithSubTypes("Wall"))
	other := NewCreature("Other", "", 1, 1)
	for _, card := range []Card{wall1, wall2, other} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall1.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall2.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: other.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("got %d triggers, want 2", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(wall1.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("clone changed original blocker")
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.FindPermanent(wall1.ID()).Counters[core.P1P1] != 1 || game.FindPermanent(wall2.ID()).Counters[core.P1P1] != 1 || game.FindPermanent(other.ID()).Counters[core.P1P1] != 0 {
			t.Fatal("wrong per-blocker capture")
		}
	}
}

func TestBlockedByPatternAbsentBlocker(t *testing.T) {
	for _, change := range []string{"returned", "phased"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			blocker := BindPermanent("blocker")
			watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(BecomesBlockedBy(This).WithBlocker(blocker), AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
			g.PutOnBattlefield(wall, b.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall.ID(), PlayerID: a.PlayerID()})
			if change == "returned" {
				g.RemoveFromBattlefield(g.FindPermanent(wall.ID()))
				g.PutOnBattlefield(wall, b.PlayerID())
			} else {
				g.PhaseOut(wall.ID())
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if change == "returned" && g.FindPermanent(wall.ID()).Counters[core.P1P1] != 0 {
				t.Fatal("old trigger changed returned blocker")
			}
		})
	}
}

func TestBlockedByPatternCombatAndSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(BecomesBlockedBy(This).OfSubtype("Wall").WithBlocker(blocker), AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
	g.PutOnBattlefield(wall, b.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: watcher.ID()}})
	if got := g.FindPermanent(wall.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("real combat did not capture blocker: %d", got)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall.ID(), PlayerID: a.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.FindPermanent(wall.ID()).Counters[core.P1P1]; got != 2 {
		t.Fatalf("source departure lost event controller or blocker: %d", got)
	}
}

func TestBlockedByPatternStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(BecomesBlockedBy(This).WithBlocker(blocker), AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
	g.PutOnBattlefield(wall, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("could not copy trigger")
	}
	g.ResolveStack()
	if got := g.FindPermanent(wall.ID()).Counters[core.P1P1]; got != 2 {
		t.Fatalf("copy lost blocker capture: %d", got)
	}
}

func TestBlockedByPatternValidation(t *testing.T) {
	for _, pattern := range []EventPattern{BecomesBlockedBy(This).OfSubtype(""), BecomesBlockedBy(This).OfSubtype("Island"), BecomesBlockedBy(This).WithBlocker(nil), BecomesBlockedBy(This).WithBlocker(BindPermanent(""))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid blocked-by pattern accepted")
		}
	}
	blocker := BindPermanent("unbound")
	card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(BecomesBlockedBy(This), AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("unbound blocker accepted: %v", err)
	}
}

func TestBlockedByPatternWithoutSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("non-Wall blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(
		BecomesBlockedBy(This).WithoutSubtype("Wall").WithBlocker(blocker),
		AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
	other := NewCreature("Other", "", 1, 4, WithSubTypes("Human"))
	g.PutOnBattlefield(wall, b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: watcher.ID()}, {BlockerID: other.ID(), AttackerID: watcher.ID()}})
	if got := g.FindPermanent(other.ID()).Counters[core.P1P1]; got != 1 {
		t.Fatalf("non-Wall blocker got %d counters, want 1", got)
	}
	if got := g.FindPermanent(wall.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("Wall blocker got %d counters, want 0", got)
	}
}

func TestBlockedByPatternWithoutSubtypeValidation(t *testing.T) {
	patterns := []EventPattern{
		BecomesBlockedBy(This).WithoutSubtype(""),
		BecomesBlockedBy(This).WithoutSubtype("Island"),
		BecomesBlockedBy(This).WithoutSubtype("Wall").WithoutSubtype("Orc"),
		BecomesBlockedBy(This).OfSubtype("Orc").WithoutSubtype("Wall"),
	}
	for _, pattern := range patterns {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid blocked-by exclusion accepted")
		}
	}
}

func TestBlockedByPatternWithoutSubtypeEventCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(
		BecomesBlockedBy(This).WithoutSubtype("Wall").WithBlocker(blocker),
		AddCountersTo(From(blocker), core.P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	human := NewCreature("Human", "", 1, 4, WithSubTypes("Human"))
	g.PutOnBattlefield(human, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: human.ID(), PlayerID: a.PlayerID()})
	human.AddSubType("Wall")
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("got %d triggers, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("could not copy trigger")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.FindPermanent(human.ID()).Counters[core.P1P1]; got != 0 {
		t.Fatalf("clone changed original blocker: %d", got)
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if got := game.FindPermanent(human.ID()).Counters[core.P1P1]; got != 2 {
			t.Fatalf("event-time match or copy lost: %d", got)
		}
	}
	g.PhaseOut(human.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: human.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 0 {
		t.Fatalf("phased-out blocker produced %d triggers", got)
	}
}
