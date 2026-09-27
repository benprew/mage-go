package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerDamageSourceCaptureLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	source := BindDamageSource("source")
	watcher := NewCreature("Watcher", "", 1, 3, WithKeyword(core.Lifelink), WithAbility(WhenEvent(
		DealsDamageToPlayer(This, player).WithSource(source),
		DealDamageFrom(FromDamageSource(source), FromPlayer(player), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage made a trigger")
	}
	g.DealDamageToPlayer(b, 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	old := g.FindPermanent(watcher.ID())
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 17 {
			t.Fatalf("source capture lost: life = %d", game.GetPlayer(b.PlayerID()).Life())
		}
		if game.GetPlayer(a.PlayerID()).Life() != 23 {
			t.Fatalf("old source controller lost lifelink: life = %d", game.GetPlayer(a.PlayerID()).Life())
		}
	}
}

func TestPlayerDamageSourceDeclaration(t *testing.T) {
	player := BindPlayer("damaged")
	source := BindDamageSource("source")
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		DealsDamageToPlayer(This, player).WithSource(source),
		DealDamageFrom(FromDamageSource(source), FromPlayer(player), Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(DealsDamageToPlayer(This, player).WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
	for _, pattern := range []EventPattern{
		DealsDamageToPlayer(This, player).WithSource(nil),
		DealsDamageToPlayer(This, player).WithSource(BindDamageSource("")),
		DealsDamageToPlayer(This, player).WithSource(source).WithSource(source),
	} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid source capture passed registration")
		}
	}
	bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
		DealsDamageToPlayer(This, player), DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound source: %v", err)
	}
}
