package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSacrificeSubtypeResultComposition(t *testing.T) {
	land := dsl.BindPermanent("chosen land")
	wasIsland := dsl.BindNumber("was Island")
	body := dsl.Sequence(
		dsl.ChoosePermanent(dsl.You, dsl.ControlledPermanents(dsl.IsLand), land),
		dsl.Sacrifice(dsl.From(land), dsl.SacrificeSubtypeInto("Island", wasIsland)),
		dsl.IfPositive(dsl.FromNumber(wasIsland), dsl.DealDamageFrom(dsl.This, dsl.You, dsl.Fixed(3))),
	)
	card := dsl.NewCreature("Synthetic", "", 2, 2, dsl.WithAbility(dsl.At(dsl.YourUpkeep, body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
