package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentLandTapBindingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		returnLand bool
		phaseLand  bool
		want       int
	}{
		{"present", false, false, 1},
		{"returned", true, false, 0},
		{"phased", false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			landBinding := BindPermanent("tapped land")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype("Forest").WithLand(landBinding),
				AddCountersTo(From(landBinding), core.Charge, Fixed(1)))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			land := NewLand("Forest", WithSubTypes("Forest"))
			g.PutOnBattlefield(land, b.PlayerID())
			g.TapPermanent(g.FindPermanent(land.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack=%d, want 1", g.StackSize())
			}
			if tc.returnLand {
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				g.PutOnBattlefield(land, b.PlayerID())
			}
			if tc.phaseLand {
				g.PhaseOut(land.ID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if got := g.FindPermanentIncludingPhased(land.ID()).Counters[core.Charge]; got != 0 {
				t.Fatalf("clone changed original: %d counters", got)
			}
			g.ResolveStack()
			if got := int(g.FindPermanentIncludingPhased(land.ID()).Counters[core.Charge]); got != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestOpponentLandTapBindingDeclaration(t *testing.T) {
	land := BindPermanent("tapped land")
	valid := NewEnchantment("Valid", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped().WithLand(land), AddCountersTo(From(land), core.Charge, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []opponentLandTappedPattern{
		OpponentLandBecomesTapped().WithLand(nil),
		OpponentLandBecomesTapped().WithLand(BindPermanent("")),
		OpponentLandBecomesTapped().WithLand(land).WithLand(land),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid land binding")
		}
	}
	unbound := NewEnchantment("Unbound", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped(), AddCountersTo(From(land), core.Charge, Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound land read")
	}
}

func TestOpponentLandTapBindingPendingCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	landBinding := BindPermanent("tapped land")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped().WithLand(landBinding),
		AddCountersTo(From(landBinding), core.Charge, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Forest", WithSubTypes("Forest"))
	g.PutOnBattlefield(land, b.PlayerID())
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending stack=%d, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if got := g.FindPermanent(land.ID()).Counters[core.Charge]; got != 2 {
		t.Fatalf("counters=%d, want 2", got)
	}
}
