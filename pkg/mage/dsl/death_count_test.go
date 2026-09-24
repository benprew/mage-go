package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreatureDeathCountComposition(t *testing.T) {
	var value = dsl.CreaturesDiedThisTurn()
	member := dsl.BindPermanent("member")
	bodies := []dsl.ResolvingEffect{
		dsl.GainLife(dsl.You, value),
		dsl.Sequence(dsl.GainLife(dsl.You, value), dsl.Draw(dsl.You, dsl.Fixed(1))),
		dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), member, dsl.AddCounters(dsl.From(member), core.Corpse, value)),
	}
	for _, body := range bodies {
		cards := []dsl.Card{
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.At(dsl.EachEndStep, body))),
			dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
			dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Count", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		}
		for _, card := range cards {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
	info := dsl.InspectDefinition(value)
	if info.Role != "creature-death-count" || info.Text != value.Text() || len(info.Children) != 0 {
		t.Fatalf("metadata = %+v", info)
	}
}
