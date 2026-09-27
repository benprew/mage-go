package mage

import (
	"strings"
	"testing"
)

func TestPlayerLosesLifeCapturesEachCompletedLoss(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("loser"), BindNumber("lost")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerLosesLife(player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PlayerLoseLife(a, 2)
	g.PlayerLoseLife(b, 3)
	g.PlayerLoseLife(b, 0)
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack = %d, want 3", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerLosesLifeSimultaneousAndSourceLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("loser"), BindNumber("lost")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerLosesLife(player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, Fixed(1)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 21 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestPlayerLosesLifeDeclaration(t *testing.T) {
	player, amount := BindPlayer("loser"), BindNumber("lost")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"valid", PlayerLosesLife(player).WithAmount(amount), ""},
		{"nil player", PlayerLosesLife(nil), "non-nil"},
		{"nil amount", PlayerLosesLife(player).WithAmount(nil), "non-nil"},
		{"empty player", PlayerLosesLife(BindPlayer("")), "name"},
		{"duplicate name", PlayerLosesLife(player).WithAmount(BindNumber("loser")), "duplicate"},
		{"repeated", PlayerLosesLife(player).WithAmount(amount).WithAmount(amount), "one amount"},
		{"missing", PlayerLosesLife(player), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern,
				GainLifeFor(FromPlayer(player), FromNumber(amount)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v, want %q", err, tc.want)
			}
		})
	}
	if got := InspectDefinition(PlayerLosesLife(player).WithAmount(amount)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}
