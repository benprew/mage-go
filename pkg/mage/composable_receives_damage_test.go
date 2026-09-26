package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestReceivesDamageCapturesActualAmountAndSourceController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	victim := NewCreature("Victim", "", 2, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	g.PutOnBattlefield(victim, a.PlayerID())
	source := NewCreature("Source", "", 2, 2)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 22 {
			t.Fatalf("source controller life = %d", got)
		}
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 20 {
			t.Fatalf("other life = %d", got)
		}
	}
}

func TestReceivesDamageAbsentPhasedAndPrevented(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	victim := NewCreature("Victim", "", 2, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	g.PutOnBattlefield(victim, a.PlayerID())
	source := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 2)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	permanent := g.FindPermanent(victim.ID())
	permanent.PhasedOut = true
	g.DealDamageToPermanent(permanent, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased victim triggered")
	}
	permanent.PhasedOut = false
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
	g.PutOnBattlefield(victim, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("departed victim triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if got := b.Life(); got != 21 {
		t.Fatalf("source controller life = %d", got)
	}
}

func TestReceivesDamageDeclaration(t *testing.T) {
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"valid", ReceivesDamage(This, amount).WithSourceController(controller), ""},
		{"nil amount", ReceivesDamage(This, nil).WithSourceController(controller), "non-nil"},
		{"empty amount", ReceivesDamage(This, BindNumber("")).WithSourceController(controller), "name"},
		{"nil controller", ReceivesDamage(This, amount).WithSourceController(nil), "non-nil"},
		{"empty controller", ReceivesDamage(This, amount).WithSourceController(BindPlayer("")), "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Test", "", 2, 5, WithAbility(WhenEvent(tc.pattern, GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(ReceivesDamage(This, amount).WithSourceController(controller)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
	badZone := NewInstant("Instant", "", WithAbility(WhenEvent(ReceivesDamage(This, amount), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(badZone.Name(), badZone); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("bad zone validation = %v", err)
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(Enters(This), GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound validation = %v", err)
	}
}

func TestReceivesDamageSimultaneousSources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	victim := NewCreature("Victim", "", 1, 8, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	g.PutOnBattlefield(victim, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DealSimultaneousDamage(
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 2, g.CaptureDamageSource(first.ID())),
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 3, g.CaptureDamageSource(second.ID())),
	)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if g.CopyStackObjectDirect(obj, uuid.Nil, false) == nil {
			t.Fatal("copy failed")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 24 || game.GetPlayer(b.PlayerID()).Life() != 26 {
			t.Fatalf("captured controllers and amounts = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}
