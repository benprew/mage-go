package mage

import (
	"strings"
	"testing"
)

func TestDealsDamageCapturesPlayerRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(player),
		GainLifeFor(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("capture or clone changed recipient")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatalf("life = %d", b.Life())
	}
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("permanent recipient supplied a player")
	}
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(b, 1, g.CaptureDamageSource(source.ID())),
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, g.CaptureDamageSource(source.ID())),
	)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("batch triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("batch lost player recipient or used permanent as player")
	}
	g.PhaseOut(source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
}

func TestDealsDamagePlayerRecipientDeclaration(t *testing.T) {
	bound := BindPlayer("recipient")
	for _, pattern := range []EventPattern{
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(nil),
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(BindPlayer("")),
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(bound).WithRecipientPlayer(bound),
		DealsDamageToPlayer(This, bound).WithRecipientPlayer(bound),
	} {
		card := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid recipient binding")
		}
	}
	card := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(DealsDamage(This, BindNumber("amount")), GainLifeFor(FromPlayer(bound), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("validation = %v", err)
	}
}

func TestDealsDamagePlayerRecipientFollowsFinalAction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("recipient")
	source := NewCreature("Source", "", 2, 2, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(player),
		GainLifeFor(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("redirected permanent damage retained a player")
	}
}
