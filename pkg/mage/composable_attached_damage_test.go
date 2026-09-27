package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestAttachedDamageEventFinalAmountsAndLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("dealt")
	aura := NewAura("Link", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, amount), GainLifeFor(You, FromNumber(amount)))))
	aura.SetOwner(a.PlayerID())
	host := NewCreature("Host", "", 2, 8)
	host.SetOwner(b.PlayerID())
	victim := NewCreature("Victim", "", 2, 12)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(b, 2, victim.ID())
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, host.ID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, host.ID())
	g.DealDamageToPlayer(b, 4, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d, want 2", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, b.PlayerID())
	g.Attach(aura.ID(), victim.ID())
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 27 {
			t.Fatalf("old controller life = %d, want 27", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 14 {
			t.Fatalf("recipient life = %d, want 14", got)
		}
	}
}

func TestAttachedDamageEventTracksAttachmentAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("damage")
	aura := NewAura("Link", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, n), GainLifeFor(You, FromNumber(n)))))
	host := NewCreature("Host", "", 2, 8)
	other := NewCreature("Other", "", 2, 8)
	for _, card := range []Card{aura, host, other} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.Attach(aura.ID(), other.ID())
	g.DealDamageToPlayer(b, 2, host.ID())
	g.DealDamageToPlayer(b, 3, other.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 14 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

func TestAttachedDamageEventDeclaration(t *testing.T) {
	n := BindNumber("amount")
	good := NewAura("Good", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, n), GainLifeFor(You, FromNumber(n)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		NewAura("Nil", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, nil), GainLifeFor(You, Fixed(1))))),
		NewAura("Unnamed", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, BindNumber("")), GainLifeFor(You, Fixed(1))))),
		NewAura("Wrong source", "", WithAbility(WhenEvent(DealsDamage(From(BindPermanent("other")), n), GainLifeFor(You, FromNumber(n))))),
		NewInstant("Spell", "", Spell(GainLifeFor(You, FromNumber(n)))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}

func TestAttachedDamageEventStackCopyKeepsAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("damage")
	aura := NewAura("Link", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, n), GainLifeFor(You, FromNumber(n)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPlayer(b, 2, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("copy life = %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}
