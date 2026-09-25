package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellStepComposition(t *testing.T) {
	var timing = dsl.OnlyDuring(core.BeginCombat, core.DeclareAttackers)
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	ability := dsl.Spell(dsl.Until(core.EndOfTurn, dsl.CantBeBlocked(target)), dsl.WithTargets(target), timing)
	card := dsl.NewInstant("Timed spell", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := dsl.InspectDefinition(ability)
	if len(info.Children) == 0 || info.Children[0].Role != "spell-step" || info.Children[0].Text != "Begin Combat or Declare Attackers" {
		t.Fatalf("timing inspection = %+v", info)
	}
}
