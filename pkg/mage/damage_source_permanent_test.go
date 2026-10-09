package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourcePermanentValidation(t *testing.T) {
	for _, state := range []string{"valid", "nil", "unnamed", "duplicate", "recipient-alias", "combat", "opponent"} {
		t.Run(state, func(t *testing.T) {
			player, source := BindPlayer("player"), BindPermanent("source")
			pattern := DealsDamageToPlayer(AttachedPermanent, player)
			switch state {
			case "nil":
				source = nil
			case "unnamed":
				source = BindPermanent("")
			case "combat":
				pattern = DealsCombatDamageToPlayer(This, player)
			case "opponent":
				pattern = DealsDamageToOpponent(This, player)
			}
			pattern = pattern.WithSourcePermanent(source)
			if state == "duplicate" {
				pattern = pattern.WithSourcePermanent(source)
			}
			if state == "recipient-alias" {
				pattern = DealsDamage(This, BindNumber("amount")).WithPermanent(source).WithSourcePermanent(source)
			}
			card := NewAura("Watcher", "", WithAbility(WhenEvent(pattern, GainLifeFor(ControllerOfWithLKI(From(source)), Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if (err == nil) != (state == "valid") {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}

func TestDamageSourcePermanentIdentity(t *testing.T) {
	for _, state := range []string{"live-control", "return", "phase", "reattach"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player, source, amount := BindPlayer("player"), BindPermanent("source"), BindNumber("amount")
			aura := g.PutOnBattlefield(NewAura("Watcher", "", WithAbility(WhenEvent(
				DealsDamageToPlayer(AttachedPermanent, player).WithSourcePermanent(source).WithAmount(amount),
				GainLifeFor(ControllerOfWithLKI(From(source)), FromNumber(amount))))), a.PlayerID())
			card := NewCreature("Host", "", 1, 9)
			host := g.PutOnBattlefield(card, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.DealDamageToPlayer(a, 2, host.ID())
			wantA, wantB := 18, 22
			switch state {
			case "live-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				wantA, wantB = 20, 20
			case "return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(host.ID())
			case "reattach":
				other := g.PutOnBattlefield(NewCreature("Other", "", 1, 9), a.PlayerID())
				g.Attach(aura.ID(), other.ID())
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestDamageSourcePermanentDistinctRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source, recipient, amount := BindPermanent("source"), BindPermanent("recipient"), BindNumber("amount")
	card := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamage(AttachedPermanent, amount).WithSourcePermanent(source).WithPermanent(recipient),
		Sequence(GainLifeFor(ControllerOfWithLKI(From(source)), FromNumber(amount)), GainLifeFor(ControllerOfWithLKI(From(recipient)), Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	aura := g.PutOnBattlefield(card, a.PlayerID())
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 9), b.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 9), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPermanent(target, 2, host.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 22 {
		t.Fatal("source and recipient bindings shared")
	}
}
