package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedDiscardEntireHandComposition(t *testing.T) {
	player := dsl.TargetPlayer("player")
	saved := dsl.BindPlayer("damaged")
	permanent := dsl.BindPermanent("permanent")
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.DiscardEntireHand(player), dsl.WithTargets(player)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.DiscardEntireHand(dsl.You), dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.DealsDamageToOpponent(dsl.This, saved), dsl.DiscardEntireHand(dsl.FromPlayer(saved))))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Discard", dsl.DiscardEntireHand(player), dsl.WithTargets(player)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), permanent, dsl.DiscardEntireHand(dsl.ControllerOf(dsl.From(permanent))))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
