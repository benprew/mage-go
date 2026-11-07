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

func TestActivatedAbilitySupportsGraveyardActivationZoneDirectly(t *testing.T) {
	g := newPriorityTestGame()
	g.turns.SetStep(PrecombatMain)
	player := g.players[0]
	card := NewCreature("Reassembling", "{1}{B}", 1, 1,
		WithAbility(NewActivated(ExileSelfFromGraveyardCost(), GainLife(2), WithSorcerySpeed(), WithActivationZone(ZoneGraveyard))),
	)
	player.AddToGraveyard(card)
	cardID, idx, ok := g.FindGraveyardActivatableCard(player.PlayerID(), "Reassembling")
	if !ok || cardID != card.ID() || idx != 0 {
		t.Fatalf("expected graveyard-zone ActionDefinition to be discoverable, got ok=%v id=%v idx=%d", ok, cardID, idx)
	}
	if err := g.ActivateGraveyardAbility(player.PlayerID(), card.ID(), idx, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if player.Life() != 22 {
		t.Fatalf("expected life 22 after graveyard ability resolved, got %d", player.Life())
	}
}
