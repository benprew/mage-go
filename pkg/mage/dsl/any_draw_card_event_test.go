package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnyDrawCardEventPublicDeclaration(t *testing.T) {
	drawer, drawn := dsl.BindPlayer("drawer"), dsl.BindCard("drawn")
	for _, pattern := range []dsl.EventPattern{
		dsl.PlayerDrawsCard(drawer).WithCard(drawn),
		dsl.OpponentDrawsCard(drawer).WithCard(drawn),
	} {
		card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern,
			dsl.IfCardHasType(dsl.FromCard(drawn), core.TypeLand,
				dsl.GainLife(dsl.FromPlayer(drawer), dsl.Fixed(1))))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
