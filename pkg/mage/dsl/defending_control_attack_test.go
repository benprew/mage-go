package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDefendingPlayerControlAttackPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Attacker", "", 2, 2,
		dsl.WithAbility(dsl.Static(dsl.CanAttackOnlyIfDefendingPlayerControls(dsl.And(dsl.IsLand, dsl.HasSubType("Island"))))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
