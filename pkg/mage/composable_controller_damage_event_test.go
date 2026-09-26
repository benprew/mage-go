package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerIsDealtDamageActualAmountAndControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 5, attacker.ID())
	g.DealDamageToPlayer(b, 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
		t.Fatalf("pending stack = %d", g.StackSize())
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatalf("life after captured damage = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
		game.DealDamageToPlayer(a, 1, attacker.ID())
		game.DealDamageToPlayer(b, 2, attacker.ID())
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatalf("pending after control change = %d", game.StackSize())
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatalf("life after control change = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestControllerIsDealtDamageReplacementAbsenceAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 1, 4)
	g.PutOnBattlefield(victim, a.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: a.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayer(a, 2, victim.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("redirected damage matched player recipient")
	}
	g.effects.replacements = nil
	g.PlayerLoseLife(a, 1)
	g.DealDamageToPlayer(a, 0, victim.ID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 2, victim.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("life loss, zero, or prevented damage triggered")
	}
	permanent := g.FindPermanent(source.ID())
	permanent.PhasedOut = true
	g.DealDamageToPlayer(a, 1, victim.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	permanent.PhasedOut = false
	g.DealDamageToPlayer(a, 1, victim.ID())
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending source departure stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 18 {
		t.Fatalf("captured trigger controller life = %d", a.Life())
	}
}

func TestControllerIsDealtDamageRegistration(t *testing.T) {
	amount := BindNumber("amount")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewEnchantment("Valid", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount))))), ""},
		{"nil", NewEnchantment("Nil", "", WithAbility(WhenEvent(ControllerIsDealtDamage(nil), GainLifeFor(You, Fixed(1))))), "non-nil"},
		{"empty", NewEnchantment("Empty", "", WithAbility(WhenEvent(ControllerIsDealtDamage(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"unbound", NewEnchantment("Unbound", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, FromNumber(amount))))), "producer"},
		{"other ability", NewEnchantment("Other", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, Fixed(1)))), WithAbility(WhenEvent(Enters(This), GainLifeFor(You, FromNumber(amount))))), "producer"},
		{"bad zone", NewInstant("Instant", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount))))), "battlefield"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(ControllerIsDealtDamage(amount)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}

func TestControllerIsDealtDamageSimultaneousCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEventModes(
		ControllerIsDealtDamage(amount),
		ModeOf("Gain", GainLifeFor(You, FromNumber(amount))),
		ModeOf("Draw", DrawFor(You, Fixed(1))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 1, 4)
	second := NewCreature("Second", "", 1, 4)
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 2, g.CaptureDamageSource(first.ID())),
		PlayerDamageAssignment(a, 3, g.CaptureDamageSource(second.ID())),
	)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("damage events = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatalf("copied captured amounts = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}
