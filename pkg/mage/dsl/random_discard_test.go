package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedRandomDiscardComposition(t *testing.T) {
	player := dsl.TargetPlayer("player")
	body := dsl.DiscardAtRandom(player, dsl.ValueAtResolution(dsl.XValue()))
	for _, card := range []mage.Card{
		dsl.NewSorcery("Spell", "{X}{B}", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(player)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(player)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(player)))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Discard", body, dsl.WithTargets(player)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
