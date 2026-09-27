package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestReceivesDamageSourceCaptureAfterDepartureAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("amount")
	source := BindDamageSource("source")
	victim := NewCreature("Victim", "", 2, 7, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, FromNumber(amount)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	card := NewCreature("Source", "", 2, 2, WithKeyword(core.Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	captured := g.CaptureDamageSource(card.ID())
	other := g.PutOnBattlefield(NewCreature("Other Source", "", 1, 2), b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealSimultaneousDamage(
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 2, captured),
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, g.CaptureDamageSource(other.ID())),
	)
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("saved source damage: A=%d B=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
		if game.FindPermanent(card.ID()).incarnationID != returned.incarnationID {
			t.Fatal("returned source changed")
		}
	}
}

func TestReceivesDamageSourceCaptureAfterSpellLeavesStack(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("spell source")
	victim := NewCreature("Victim", "", 1, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, BindNumber("amount")).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	spell := NewInstant("Spell", "", nil)
	spell.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: spell, Controller: a.PlayerID(), SourceID: spell.ID()})
	captured := g.CaptureDamageSource(spell.ID())
	end()
	if !captured.Valid() {
		t.Fatal("spell source was not captured")
	}
	g.DealDamageToPermanentFromSource(g.FindPermanent(victim.ID()), 1, captured)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatalf("spell source did not survive stack departure: life = %d", b.Life())
	}
}

func TestReceivesDamageSourceCapturePreventionPhasingAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	victim := NewCreature("Victim", "", 1, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, BindNumber("amount")).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage created a trigger")
	}
	g.PhaseOut(victim.ID())
	g.DealDamageToPermanent(g.FindPermanentIncludingPhased(victim.ID()), 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased recipient created a trigger")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.PhaseOut(attacker.ID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 18 {
		t.Fatalf("copied captures lost phased source: life = %d", b.Life())
	}
}

func TestReceivesDamageSourceCaptureAfterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	victim := NewCreature("Victim", "", 1, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, BindNumber("amount")).WithSource(source),
		DealDamageFrom(FromDamageSource(source), ActivePlayer, Fixed(1)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(attacker.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("redirected damage triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatalf("redirected source was not captured: A=%d B=%d", a.Life(), b.Life())
	}
}

func TestReceivesDamageSourceDeclaration(t *testing.T) {
	source := BindDamageSource("source")
	amount := BindNumber("amount")
	good := NewCreature("Good", "", 1, 3, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, FromNumber(amount)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(ReceivesDamage(This, amount).WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
	for _, pattern := range []EventPattern{
		ReceivesDamage(This, amount).WithSource(nil),
		ReceivesDamage(This, amount).WithSource(BindDamageSource("")),
		ReceivesDamage(This, amount).WithSource(source).WithSource(source),
	} {
		bad := NewCreature("Bad", "", 1, 3, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid capture passed registration")
		}
	}
	bad := NewCreature("Bad", "", 1, 3, WithAbility(WhenEvent(ReceivesDamage(This, amount), DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound source: %v", err)
	}
}
