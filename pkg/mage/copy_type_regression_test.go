package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCopyAdditionalTypeIsUnique(t *testing.T) {
	g := newPriorityTestGame()
	card := NewArtifact("Artifact", "{0}")
	card.AddType(TypeEnchantment)
	target := g.PutOnBattlefield(card, g.players[0].PlayerID())
	receiver := NewEnchantment("Copy", "{0}")
	g.resolution.SetResolvingCard(receiver)
	if err := CloneTarget(TypeEnchantment).Apply(&EffectContext{Game: g, Targets: []uuid.UUID{target.ID()}}); err != nil {
		t.Fatal(err)
	}
	if len(receiver.Types()) != 2 {
		t.Fatalf("duplicate types: %v", receiver.Types())
	}
}
