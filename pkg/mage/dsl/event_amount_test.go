package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedDamageEventAmount(t *testing.T) {
	for _, variant := range []int{0, 1, 2} {
		player, amount := dsl.BindPlayer("damaged player"), dsl.BindNumber("damage")
		pattern := dsl.DealsDamageToPlayer(dsl.This, player).WithAmount(amount)
		if variant == 1 {
			pattern = dsl.DealsDamageToOpponent(dsl.This, player).WithAmount(amount)
		}
		if variant == 2 {
			pattern = dsl.DealsDamage(dsl.This, amount)
		}
		target := dsl.TargetPlayer("mill player")
		body := dsl.Mill(target, dsl.FromNumber(amount))
		card := dsl.NewCreature("Source", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(pattern, body, dsl.WithTargets(target))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var capture, value bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			capture = capture || n.Role == "event-amount-capture"
			value = value || n.Role == "resolution-value" && n.Text == "damage"
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(card.Abilities()[0]))
		if !capture || !value {
			t.Fatal("missing amount definition metadata")
		}
	}
}
