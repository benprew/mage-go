package mage

import (
	"strings"
	"testing"
)

func TestPlayerGainsLifeCapturesActualEventAndEachPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("amount")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerGainsLife(player).WithAmount(amount),
		LoseLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 3})
	g.gainLifeWithEvent(a, 1)
	g.gainLifeWithEvent(b, 1)
	g.gainLifeWithEvent(a, 0)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerGainsLifeSourceAndFrameLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("amount")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerGainsLife(player).WithAmount(amount),
		LoseLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.gainLifeWithEvent(b, 1)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.gainLifeWithEvent(b, 2)
	g.gainLifeWithEvent(a, 4)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[1], b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 21 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestPlayerGainsLifeDeclaration(t *testing.T) {
	player, amount := BindPlayer("gainer"), BindNumber("amount")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"valid", PlayerGainsLife(player).WithAmount(amount), ""},
		{"nil player", PlayerGainsLife(nil), "non-nil"},
		{"nil amount", PlayerGainsLife(player).WithAmount(nil), "non-nil"},
		{"empty", PlayerGainsLife(BindPlayer("")), "name"},
		{"duplicate", PlayerGainsLife(player).WithAmount(BindNumber("gainer")), "duplicate"},
		{"repeated", PlayerGainsLife(player).WithAmount(amount).WithAmount(amount), "one amount"},
		{"missing", PlayerGainsLife(player), "producer"},
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
	if got := InspectDefinition(PlayerGainsLife(player).WithAmount(amount)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}
