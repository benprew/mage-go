package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerDamageSourceCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, source := BindNumber("amount"), BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, FromNumber(amount)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	dealer := NewCreature("Dealer", "", 2, 2, WithKeyword(core.Lifelink))
	g.PutOnBattlefield(dealer, b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealDamageToPlayer(a, 1, dealer.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	g.DealDamageToPlayer(a, 2, dealer.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(dealer.ID()))
	g.PutOnBattlefield(dealer, a.PlayerID())
	g.PhaseOut(dealer.ID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 16 {
			t.Fatalf("damaged player life = %d", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 24 {
			t.Fatalf("old source controller life = %d", got)
		}
	}
}

func TestControllerDamageSourceDeclaration(t *testing.T) {
	amount, source := BindNumber("amount"), BindDamageSource("source")
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(ControllerIsDealtDamage(amount).WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
	for _, pattern := range []EventPattern{
		ControllerIsDealtDamage(amount).WithSource(nil),
		ControllerIsDealtDamage(amount).WithSource(BindDamageSource("")),
		ControllerIsDealtDamage(amount).WithSource(source).WithSource(source),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, DrawFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid capture passed validation")
		}
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound source: %v", err)
	}
}
