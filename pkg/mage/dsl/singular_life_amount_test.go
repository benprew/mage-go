package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedSingularLifeAmountComposition(t *testing.T) {
	for _, constructor := range []func(dsl.PlayerRef, dsl.ResolutionValue) dsl.ResolvingEffect{dsl.GainLife, dsl.LoseLife} {
		saved := dsl.BindPermanent("amount")
		effect := constructor(dsl.You, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved))))
		body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), effect)
		player := dsl.TargetPlayer("player")
		target := dsl.TargetPermanent("amount", dsl.IsCreature)
		for _, card := range []mage.Card{
			dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(constructor(player, dsl.ToughnessOf(target)), dsl.WithTargets(player, target)))),
			dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
			dsl.NewCreature("Delayed", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.AtNext(dsl.EachEndStep, effect))))),
			dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, effect)))),
			dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", constructor(dsl.You, dsl.PowerOf(target)), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
}
