package mage

import (
	"strings"
	"testing"
)

func TestComposableCoinFlipBranches(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetCoinFlipResults([]bool{true, false})
	effect := FlipCoin(You, OnWin(GainLifeFor(You, Fixed(2))), OnLoss(LoseLifeFor(You, Fixed(3))))
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		for _, want := range []int{22, 19} {
			if err := ApplyEffect(game, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if got := game.GetPlayer(a.PlayerID()).Life(); got != want {
				t.Fatalf("life = %d, want %d", got, want)
			}
		}
	}
	metadata := InspectDefinition(effect)
	if len(metadata.Children) != 3 || len(metadata.Children[1].Children) != 1 || len(metadata.Children[2].Children) != 1 {
		t.Fatalf("missing branch metadata: %+v", metadata)
	}
}

func TestComposableCoinFlipValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		effect ResolvingEffect
		want   string
	}{
		{"empty", FlipCoin(You), "branch"},
		{"duplicate", FlipCoin(You, OnWin(GainLifeFor(You, Fixed(1))), OnWin(GainLifeFor(You, Fixed(2)))), "duplicate"},
		{"nil effect", FlipCoin(You, OnLoss(nil)), "non-nil"},
		{"nil player", FlipCoin(nil, OnLoss(GainLifeFor(You, Fixed(1)))), "player"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewSorcery("Coin", "", WithAbility(Spell(tc.effect)))
			err := ValidateCardDefinition("Coin", card)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
	value := BindPermanent("value")
	effect := Sequence(FlipCoin(You, OnWin(Snapshot(This, value))), Sacrifice(From(value)))
	if err := ValidateCardDefinition("Scope", NewCreature("Scope", "", 1, 1, WithAbility(Activate(Costs(), effect)))); err == nil {
		t.Fatal("coin branch exported a conditional assignment")
	}
}
