package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedRandomPlayerComposition(t *testing.T) {
	var saved = dsl.BindPlayer("recipient")
	var player = dsl.FromPlayer(saved)
	body := dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.Mill(player, dsl.Fixed(2)))
	for _, card := range []mage.Card{
		dsl.NewSorcery("Draw", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.Draw(player, dsl.Fixed(3)))))),
		dsl.NewSorcery("GainLife", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.GainLife(player, dsl.Fixed(3)))))),
		dsl.NewSorcery("Discard", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.EachPlayer, saved), dsl.Discard(player, dsl.Fixed(1)))))),
		dsl.NewSorcery("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Random", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewSorcery("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)))),
		dsl.NewSorcery("Delayed", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.ChooseRandomPlayer(dsl.OpponentsOf(dsl.You), saved), dsl.AtNext(dsl.EachEndStep, dsl.PreventNextDamage(player, dsl.Fixed(2))))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
