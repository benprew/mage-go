package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreatureDiesPattern(t *testing.T) {
	var pattern = dsl.CreatureDies()
	target := dsl.TargetPlayer("player")
	for _, ability := range []mage.TriggeredAbility{
		dsl.WhenEvent(pattern, dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
		dsl.WhenEventModes(pattern,
			dsl.ModeOf("Mill", dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
			dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))),
	} {
		card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var found bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			found = found || n.Role == "event-pattern" && n.Text == "a creature dies"
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(ability))
		if !found {
			t.Fatal("missing creature death pattern metadata")
		}
	}
}
func TestCreatureYouControlDiesPattern(t *testing.T) {
	var pattern = dsl.CreatureYouControlDies()
	target := dsl.TargetPlayer("player")
	for _, ability := range []mage.TriggeredAbility{
		dsl.WhenEvent(pattern, dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
		dsl.WhenEventModes(pattern,
			dsl.ModeOf("Mill", dsl.Mill(target, dsl.Fixed(1)), dsl.WithTargets(target)),
			dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))),
	} {
		card := dsl.NewCreature("Watcher", "", 1, 1, dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var found bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			found = found || n.Role == "event-pattern" && n.Text == "a creature you control dies"
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(ability))
		if !found {
			t.Fatal("missing creature death pattern metadata")
		}
	}
}
