package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestUntapEachAuthorExample(t *testing.T) {
	card := dsl.NewInstant("Land untap", "", dsl.WithAbility(dsl.Spell(
		dsl.UntapEach(dsl.ControlledPermanents(dsl.IsLand)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
