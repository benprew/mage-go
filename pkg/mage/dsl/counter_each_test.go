package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAddCountersEachAuthorExample(t *testing.T) {
	card := dsl.NewSorcery("Counter batch", "", dsl.WithAbility(dsl.Spell(
		dsl.AddCountersEach(dsl.ControlledPermanents(dsl.IsCreature), dsl.P1P1, dsl.Fixed(2)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
