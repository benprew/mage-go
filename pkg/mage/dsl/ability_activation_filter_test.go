package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAbilityActivationFilterPublicDeclaration(t *testing.T) {
	actor := dsl.BindPlayer("activator")
	card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.ActivatedAbilityEvent().FromArtifact().ByOpponent().WithActivator(actor),
		dsl.GainLife(dsl.FromPlayer(actor), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
