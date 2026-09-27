package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeathPermanentBindingKeepsDepartedDamageSource(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		dead := BindPermanent("dead")
		watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
			AnotherCreatureDies().WithPermanent(dead),
			Sequence(AddCountersTo(From(dead), core.P1P1, Fixed(1)),
				DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(2))))))
		watcher.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
			t.Fatal(err)
		}
		victim := NewCreature("Victim", "", 2, 2, WithKeyword(core.Lifelink))
		victim.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(victim, b.PlayerID())
		if simultaneous {
			g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
		} else {
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatalf("simultaneous=%v: stack=%d", simultaneous, g.StackSize())
		}
		g.PutOnBattlefield(victim, a.PlayerID())
		clone := g.Clone()
		clone.ResolveStack()
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("clone changed original")
		}
		g.ResolveStack()
		if got := g.FindPermanent(victim.ID()).Counters[core.P1P1]; got != 0 {
			t.Fatalf("returned incarnation gained %d counters", got)
		}
		if a.Life() != 18 || b.Life() != 22 {
			t.Fatalf("simultaneous=%v: life A=%d B=%d", simultaneous, a.Life(), b.Life())
		}
		if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("clone lost death capture")
		}
	}
}

func TestDeathPermanentBindingDeclaration(t *testing.T) {
	dead := BindPermanent("dead")
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithPermanent(dead), DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(good.Abilities()[0]); meta.Role != "event-trigger" || len(meta.Children[0].Children) == 0 {
		t.Fatalf("missing capture inspection: %+v", meta)
	}
	for _, pattern := range []creatureDiesPattern{
		CreatureDies().WithPermanent(nil),
		CreatureDies().WithPermanent(BindPermanent("")),
		CreatureDies().WithPermanent(dead).WithPermanent(dead),
	} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid death capture")
		}
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies(), DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound death reference")
	}
}

func TestDeathPermanentBindingStackCopyAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindPermanent("dead")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().WithPermanent(dead),
		DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn([]uuid.UUID{watcher.ID()})
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("present source did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatalf("copy lost binding: life A=%d B=%d", a.Life(), b.Life())
	}
}
