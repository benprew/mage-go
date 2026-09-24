package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedCollectionLifeAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount")
	loss := dsl.LoseLifeEach(dsl.OpponentsOf(dsl.You), dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), loss)
	player := dsl.TargetPlayer("player")
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.LoseLifeEach(dsl.OpponentsOf(player), dsl.ToughnessOf(target)), dsl.WithTargets(player, target)))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, loss)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", dsl.LoseLifeEach(dsl.EachPlayer, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
