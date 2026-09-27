package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOptionalGraveyardTargetPublicDeclaration(t *testing.T) {
	target := dsl.TargetGraveyardCard("card",
		dsl.GraveyardUpToOne(),
		dsl.GraveyardCardFilter(dsl.IsCreatureCard),
		dsl.GraveyardCardManaValueAtMost(dsl.ControllerLifeGainedThisTurn()))
	card := dsl.NewCreature("Synthetic watcher", "", 1, 1,
		dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This),
			dsl.ReturnCardToBattlefield(target, dsl.You), dsl.WithTargets(target))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
