package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentBecomesTappedCapturesTransition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	object := BindPermanent("tapped permanent")
	controller := BindPlayer("tap controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PermanentBecomesTapped().OfType(core.TypeCreature).ByOpponent().WithPermanent(object).WithController(controller),
		Sequence(AddCountersTo(From(object), core.Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	creature := NewCreature("Creature", "", 2, 2)
	g.PutOnBattlefield(creature, b.PlayerID())
	g.TapPermanent(g.FindPermanent(creature.ID()))
	g.TapPermanent(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || g.FindPermanent(creature.ID()).Counters[core.Charge] != 0 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if b.Life() != 21 || g.FindPermanent(creature.ID()).Counters[core.Charge] != 1 {
		t.Fatal("tap participants were not saved")
	}
}

func TestPermanentBecomesTappedIdentityAndSourceLifecycle(t *testing.T) {
	for _, state := range []string{"returned", "phased subject", "phased source", "source leaves", "present"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			object := BindPermanent("object")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				PermanentBecomesTapped().WithPermanent(object), AddCountersTo(From(object), core.Charge, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			card := NewArtifact("Object", "")
			g.PutOnBattlefield(card, b.PlayerID())
			if state == "phased source" {
				g.PhaseOut(watcher.ID())
			}
			if state == "phased subject" {
				g.PhaseOut(card.ID())
			}
			g.TapPermanent(g.FindPermanent(card.ID()))
			if state == "source leaves" {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			}
			g.PutTriggersOnStack()
			want := 1
			if state == "phased source" || state == "phased subject" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), want)
			}
			if state == "returned" {
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.ResolveStack()
			wantCounters := uint8(1)
			if state == "returned" || want == 0 {
				wantCounters = 0
			}
			if got := g.FindPermanentIncludingPhased(card.ID()).Counters[core.Charge]; got != wantCounters {
				t.Fatalf("counters=%d, want %d", got, wantCounters)
			}
		})
	}
}

func TestPermanentBecomesTappedDeclaration(t *testing.T) {
	object := BindPermanent("object")
	controller := BindPlayer("controller")
	valid := NewArtifact("Valid", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithPermanent(object).WithController(controller), GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{
		PermanentBecomesTapped().WithPermanent(nil),
		PermanentBecomesTapped().WithPermanent(BindPermanent("")),
		PermanentBecomesTapped().WithPermanent(object).WithPermanent(object),
		PermanentBecomesTapped().WithController(nil),
		PermanentBecomesTapped().WithController(controller).WithController(controller),
		PermanentBecomesTapped().ByOpponent().ByOpponent(),
		PermanentBecomesTapped().OfType(core.TypeCreature).OfType(core.TypeArtifact),
		PermanentBecomesTapped().OfType(core.TypeInstant),
	} {
		bad := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid tap pattern")
		}
	}
	unbound := NewArtifact("Unbound", "", WithAbility(WhenEvent(PermanentBecomesTapped(), AddCountersTo(From(object), core.Charge, Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound event participant")
	}
}

func TestPermanentBecomesTappedFilterAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("tap controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PermanentBecomesTapped().OfType(core.TypeCreature).ByOpponent().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Own", "", 1, 1), a.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Other type", ""), b.PlayerID())
	g.TapPermanent(artifact)
	if err := TapEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !first.Tapped || !second.Tapped {
		t.Fatal("batch did not tap both creatures")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 23 || a.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}
