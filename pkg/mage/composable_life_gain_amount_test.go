package mage

import (
	"strings"
	"testing"
)

func TestControllerLifeGainAmountCapturesFinalEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("gained")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerGainsLife(player).WithAmount(amount),
		LoseLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	g.gainLifeWithEvent(a, 2)
	g.gainLifeWithEvent(b, 2)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 25 {
			t.Fatalf("opponent life = %d, want 25", got)
		}
	}
}

func TestControllerLifeGainAmountKeepsEachTriggerAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("gained")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerGainsLife(player).WithAmount(amount),
		LoseLifeFor(You, FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.gainLifeWithEvent(a, 2)
	g.gainLifeWithEvent(a, 4)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	objects := g.StackObjects()
	g.CopyStackObjectDirect(objects[len(objects)-1], b.PlayerID(), false)
	g.ResolveTopOfStack()
	if got := b.Life(); got != 18 {
		t.Fatalf("copy controller life = %d, want 18", got)
	}
	g.ResolveTopOfStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("first trigger life = %d, want 24", got)
	}
	g.ResolveTopOfStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("second trigger life = %d, want 20", got)
	}
}

func TestControllerLifeGainAmountDeclaration(t *testing.T) {
	player, amount := BindPlayer("gainer"), BindNumber("gained")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"valid", ControllerGainsLife(player).WithAmount(amount), ""},
		{"nil", ControllerGainsLife(player).WithAmount(nil), "non-nil"},
		{"empty", ControllerGainsLife(player).WithAmount(BindNumber("")), "name"},
		{"duplicate", ControllerGainsLife(player).WithAmount(BindNumber("gainer")), "duplicate"},
		{"repeated", ControllerGainsLife(player).WithAmount(amount).WithAmount(amount), "one amount"},
		{"missing", ControllerGainsLife(player), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern,
				LoseLifeFor(FromPlayer(player), FromNumber(amount)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v, want %q", err, tc.want)
			}
		})
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(
		ControllerGainsLife(player).WithAmount(amount), LoseLifeFor(FromPlayer(player), FromNumber(amount)))),
		WithAbility(At(YourUpkeep, LoseLifeFor(You, FromNumber(amount)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope validation = %v", err)
	}
	if got := InspectDefinition(ControllerGainsLife(player).WithAmount(amount)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}
