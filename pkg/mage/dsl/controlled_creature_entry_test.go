package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestControlledCreatureEntryPublicDeclaration(t *testing.T) {
	power := dsl.BindNumber("entry power")
	card := dsl.NewCreature("Watcher", "", 3, 3, dsl.WithAbility(dsl.WhenEvent(
		dsl.CreatureYouControlEnters().HasKeyword(core.Flying).WithPower(power),
		dsl.GainLife(dsl.You, dsl.FromNumber(power)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
