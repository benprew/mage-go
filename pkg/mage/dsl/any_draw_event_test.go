package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerDrawsCardPublicDeclaration(t *testing.T) {
	drawer := dsl.BindPlayer("drawer")
	card := dsl.NewEnchantment("Draw watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PlayerDrawsCard(drawer), dsl.LoseLife(dsl.FromPlayer(drawer), dsl.Fixed(2)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
