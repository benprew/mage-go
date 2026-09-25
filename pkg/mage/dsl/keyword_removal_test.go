package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedKeywordRemovalComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	body := dsl.Until(dsl.EndOfTurn, dsl.RemoveKeyword(target, dsl.Forestwalk))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.Spell(body, dsl.WithTargets(target))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.RemoveKeyword(dsl.This, dsl.Forestwalk))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
