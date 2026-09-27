package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestEntryCastColorsPublicDSL(t *testing.T) {
	card := dsl.NewCreature("Entry", "{3}", 2, 2, dsl.WithAbility(
		dsl.EntersWithCounters(core.P1P1, dsl.ColorsSpentToCastEnteringPermanent())))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
