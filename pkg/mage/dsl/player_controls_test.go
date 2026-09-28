package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPlayerControlsPublicDeclaration(t *testing.T) {
	card := dsl.NewInstant("Watch the active player", "", dsl.WithAbility(dsl.Spell(
		dsl.IfState(dsl.PlayerControls(dsl.ActivePlayer, dsl.IsLand),
			dsl.GainLife(dsl.You, dsl.Fixed(1))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
