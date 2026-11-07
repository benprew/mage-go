package search

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestBadMoonHandValueUsesBlackCreatures(t *testing.T) {
	g, player, _ := makeGame()
	moon := mage.NewEnchantment("Bad Moon", "{1}{B}")
	baseline := cardHandValue(g, player, moon)
	for range 3 {
		g.AddToBattlefield(makePerm("White", "{W}", 1, 1, player.PlayerID()))
	}
	if got := cardHandValue(g, player, moon); got != baseline {
		t.Fatalf("white creatures changed Bad Moon value: got %v, want %v", got, baseline)
	}
	for range 3 {
		g.AddToBattlefield(makePerm("Black", "{B}", 1, 1, player.PlayerID()))
	}
	if got := cardHandValue(g, player, moon); got <= baseline {
		t.Fatalf("black creatures did not increase Bad Moon value: got %v, baseline %v", got, baseline)
	}
}
