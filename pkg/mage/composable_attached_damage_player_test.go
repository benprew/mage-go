package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestAttachedDamageToPlayerCapturesParticipantAndAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	amount := BindNumber("amount")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)),
	)))
	host := NewCreature("Host", "", 2, 8)
	other := NewCreature("Other", "", 2, 8)
	for _, card := range []Card{aura, host, other} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPermanent(g.FindPermanent(other.ID()), 2, host.ID())
	g.DealDamageToPlayer(b, 1, other.ID())
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, host.ID())
	g.DealDamageToPlayer(b, 3, host.ID())
	g.DealDamageToPlayer(a, 1, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d, want 2", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, b.PlayerID())
	g.Attach(aura.ID(), other.ID())
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatalf("life = %d/%d, want 20/19", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestAttachedDamageToPlayerPhasingAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("player")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, p), GainLifeFor(FromPlayer(p), Fixed(1)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("recipient life = %d, want 20", game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestAttachedDamageToPlayerRegistration(t *testing.T) {
	p := BindPlayer("player")
	n := BindNumber("amount")
	for _, pattern := range []EventPattern{
		DealsDamageToPlayer(From(BindPermanent("other")), p),
		DealsDamageToPlayer(AttachedPermanent, nil),
		DealsDamageToPlayer(AttachedPermanent, BindPlayer("")),
		DealsDamageToPlayer(AttachedPermanent, p).WithAmount(nil),
		DealsDamageToPlayer(AttachedPermanent, p).WithAmount(BindNumber("")),
	} {
		card := NewAura("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid declaration: %#v", pattern)
		}
	}
	card := NewAura("Valid", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, p).WithAmount(n), GainLifeFor(FromPlayer(p), FromNumber(n)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
