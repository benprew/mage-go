package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestObjectFilterUsesCurrentCharacteristicsAcrossZones(t *testing.T) {
	g := newPriorityTestGame()
	card := NewCreature("Creature", "{G}", 2, 3, WithSubTypes("Elf"))
	permanent := g.PutOnBattlefield(card, g.players[0].PlayerID())
	filter := ObjectAnd(ObjectHasType(TypeCreature), ObjectHasSubtype("Elf"), ObjectHasColor(Green))
	if !filter.Match(card, g) || !filter.Match(permanent, g) {
		t.Fatal("shared filter disagrees across zones")
	}
	permanent.SubTypeOverride = []string{"Goblin"}
	if filter.Match(permanent, g) || !filter.Match(card, g) {
		t.Fatal("filter ignored current battlefield characteristics")
	}
	if !filter.Cards().Match(card) || filter.Permanents().Match(permanent, g) {
		t.Fatal("filter adapter changed meaning")
	}
}
