package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentTapEventDeclaration(t *testing.T) {
	permanent := dsl.BindPermanent("tapped permanent")
	controller := dsl.BindPlayer("controller")
	card := dsl.NewEnchantment("Observer", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PermanentBecomesTapped().OfType(dsl.TypeArtifact).ByOpponent().WithPermanent(permanent).WithController(controller),
		dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
