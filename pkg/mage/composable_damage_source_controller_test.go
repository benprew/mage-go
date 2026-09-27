package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourceControllerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	controller := BindPlayer("source controller")
	amount := BindNumber("amount")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, player).ToController().WithSourceController(controller).WithAmount(amount),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(a, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 2, host.ID())
	g.DealDamageToPlayer(a, 3, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 16 || game.GetPlayer(b.PlayerID()).Life() != 26 {
			t.Fatalf("life = %d/%d, want 16/26", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestDamageSourceControllerThisAndValidation(t *testing.T) {
	player := BindPlayer("damaged")
	controller := BindPlayer("source controller")
	valid := NewCreature("Valid", "", 1, 4, WithAbility(WhenEvent(
		DealsDamageToPlayer(This, player).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	valid.SetOwner(a.PlayerID())
	g.PutOnBattlefield(valid, a.PlayerID())
	g.DealDamageToPlayer(b, 1, valid.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatalf("life = %d/%d, want 21/19", a.Life(), b.Life())
	}
	for _, tt := range []struct {
		name string
		card Card
		want string
	}{
		{"nil", NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(nil), DrawFor(You, Fixed(1))))), "controller binding"},
		{"unnamed", NewCreature("Unnamed", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(BindPlayer("")), DrawFor(You, Fixed(1))))), "name"},
		{"duplicate", NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(controller).WithSourceController(controller), DrawFor(You, Fixed(1))))), "controller binding"},
		{"same binding", NewCreature("Same", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(player), DrawFor(You, Fixed(1))))), "duplicate"},
		{"wrong pattern", NewCreature("Wrong", "", 1, 1, WithAbility(WhenEvent(DealsDamageToOpponent(This, player).WithSourceController(controller), DrawFor(You, Fixed(1))))), "DealsDamageToPlayer"},
		{"unbound", NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player), GainLifeFor(FromPlayer(controller), Fixed(1))))), "producer"},
		{"cross ability", NewCreature("Cross", "", 1, 1,
			WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(controller), DrawFor(You, Fixed(1)))),
			WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(controller), Fixed(1))))), "producer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCardDefinition(tt.card.Name(), tt.card)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
