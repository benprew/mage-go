package ai

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestImportManaMetadataInDeckEvaluation(t *testing.T) {
	land := mage.NewLand("Compiled Land", mage.WithAbility(mage.NewActivated(mage.Tap(), mage.AddMana(core.Green, 1))))
	if colors := handLandColors([]*mage.Card{land}); colors[core.Green] != 1 {
		t.Fatalf("compiled land colors=%v", colors)
	}
	creature := mage.NewCreature("Compiled Elf", "{G}", 1, 1, mage.WithAbility(mage.NewActivated(mage.Tap(), mage.AddMana(core.Green, 1))))
	if !hasManaAcceleration(creature) {
		t.Fatal("compiled mana creature is not recognized as acceleration")
	}
}
