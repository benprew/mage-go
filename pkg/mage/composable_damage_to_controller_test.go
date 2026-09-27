package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageToControllerEventFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p, n := BindPlayer("damaged"), BindNumber("amount")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, p).ToController().WithAmount(n),
		GainLifeFor(FromPlayer(p), FromNumber(n)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPlayer(b, 2, host.ID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 2, host.ID())
	g.DealDamageToPlayer(a, 3, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, b.PlayerID())
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("life = %d/%d, want 20/18", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestDamageToControllerEventCurrentControllerAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, p).ToController(),
		GainLifeFor(FromPlayer(p), Fixed(1)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(a, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.DealDamageToPlayer(a, 1, host.ID())
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatalf("life = %d/%d, want 18/21", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestDamageToControllerEventDeclaration(t *testing.T) {
	p := BindPlayer("damaged")
	valid := NewAura("Valid", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, p).ToController(), DrawFor(FromPlayer(p), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewAura("Invalid", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(From(BindPermanent("other")), p).ToController(), DrawFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("accepted invalid source")
	}
	for _, pattern := range []EventPattern{
		DealsDamageToOpponent(This, p).ToController(),
		DealsCombatDamageToPlayer(This, p).ToController(),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, DrawFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted incompatible controller filter")
		}
	}
}

func TestDamageToControllerEventSourceThis(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	source := NewCreature("Watcher", "", 1, 4, WithAbility(WhenEvent(
		DealsDamageToPlayer(This, p).ToController(), GainLifeFor(FromPlayer(p), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(b, 1, source.ID())
	g.DealDamageToPlayer(a, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatalf("life = %d/%d, want 21/19", a.Life(), b.Life())
	}
}
