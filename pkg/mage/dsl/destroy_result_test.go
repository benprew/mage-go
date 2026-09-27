package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreaturesDiedThisWayIntoPublicDeclaration(t *testing.T) {
	deaths := dsl.BindNumber("creatures died this way")
	body := dsl.Sequence(
		dsl.DestroyEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.CreaturesDiedThisWayInto(deaths)),
		dsl.DealDamageFrom(dsl.This, dsl.You, dsl.FromNumber(deaths)),
	)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatalf("%s: %v", card.Name(), err)
		}
	}
}
