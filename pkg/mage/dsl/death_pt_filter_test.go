package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDeathPowerOrToughnessFilterPublic(t *testing.T) {
	opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
	card := dsl.NewCreature("Watcher", "", 2, 2, dsl.WithAbility(dsl.WhenEvent(
		dsl.CreatureYouControlDies().PowerOrToughnessAtMost(1),
		dsl.Sequence(dsl.LoseLife(opponent, dsl.Fixed(2)), dsl.GainLife(dsl.You, dsl.Fixed(2))),
		dsl.WithTargets(opponent))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
