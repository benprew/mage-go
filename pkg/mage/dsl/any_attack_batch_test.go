package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnyAttackBatchPublicDeclaration(t *testing.T) {
	count := dsl.BindNumber("declared attackers")
	pattern := dsl.OneOrMoreCreaturesAttack().WithCount(count)
	card := dsl.NewEnchantment("Attack observer", "", dsl.WithAbility(dsl.WhenEvent(
		pattern, dsl.GainLife(dsl.You, dsl.FromNumber(count)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if info := dsl.InspectDefinition(pattern); info.Role != "event-pattern" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
}
