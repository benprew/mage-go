package mage

import (
	"strings"
	"testing"
)

func TestDealsDamageAnyRecipientCapturesSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
		DealsDamage(This, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, FromNumber(amount)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.DealDamageToPlayer(b, 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("player damage triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 19 {
			t.Fatalf("departed source damage = %d", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 18 {
			t.Fatalf("copied trigger damage = %d", got)
		}
	}
}

func TestDealsDamageAnyRecipientSourceDeclaration(t *testing.T) {
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(DealsDamage(This, amount).WithSource(saved)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
	for _, pattern := range []EventPattern{
		DealsDamage(This, amount).WithSource(nil),
		DealsDamage(This, amount).WithSource(BindDamageSource("")),
		DealsDamage(This, amount).WithSource(saved).WithSource(saved),
	} {
		bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid source capture passed validation")
		}
	}
	bad := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(DealsDamage(This, amount), DealDamageFrom(FromDamageSource(saved), You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound source: %v", err)
	}
}

func TestDealsDamageAnyRecipientSourceBatchAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
		DealsDamage(This, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, FromNumber(amount)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	g.PutOnBattlefield(victim, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	if err := ApplyEffect(g, DealDamageEach(This,
		DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(1)),
		watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("batch triggers = %d", g.StackSize())
	}
	g.PhaseOut(watcher.ID())
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 16 {
		t.Fatalf("life after three captured sources = %d", got)
	}
}

func TestAttachedPermanentAnyDamageSourceCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	aura := NewAura("Link", "", WithAbility(WhenEvent(
		DealsDamage(AttachedPermanent, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, FromNumber(amount)))))
	host := NewCreature("Host", "", 2, 8)
	victim := NewCreature("Victim", "", 2, 8)
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("attached source triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.PhaseOut(aura.ID())
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 18 {
		t.Fatalf("saved host damage = %d", got)
	}
}
