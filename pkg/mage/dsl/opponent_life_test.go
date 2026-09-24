package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedOpponentLifeComposition(t *testing.T) {
	body := dsl.Sequence(dsl.LoseLifeEach(dsl.OpponentsOf(dsl.You), dsl.Fixed(3)), dsl.GainLife(dsl.You, dsl.Fixed(3)))
	player := dsl.TargetPlayer("player")
	for _, card := range []mage.Card{
		dsl.NewSorcery("Life", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewSorcery("Target", "", dsl.WithAbility(dsl.Spell(dsl.LoseLifeEach(dsl.OpponentsOf(player), dsl.Fixed(3)), dsl.WithTargets(player)))),
		dsl.NewSorcery("Draw", "", dsl.WithAbility(dsl.Spell(dsl.DrawEach(dsl.OpponentsOf(dsl.You), dsl.Fixed(1))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
