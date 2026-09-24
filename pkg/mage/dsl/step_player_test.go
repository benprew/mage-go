package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTurnStepPlayerComposition(t *testing.T) {
	player := dsl.BindPlayer("step player")
	pattern := dsl.TurnStepBegins(dsl.EachUpkeep, player)
	body := dsl.Sequence(dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)),
		dsl.AtNext(dsl.EachEndStep, dsl.PreventNextDamage(dsl.FromPlayer(player), dsl.Fixed(2))))
	for _, ability := range []dsl.TriggeredAbility{
		dsl.WhenEvent(pattern, body),
		dsl.WhenEventModes(pattern, dsl.ModeOf("Life", body), dsl.ModeOf("Draw", dsl.Draw(dsl.FromPlayer(player), dsl.Fixed(1)))),
		dsl.WhenEvent(pattern, dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)),
	} {
		card := dsl.NewEnchantment("Watcher", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(pattern)
	if info.Role != "event-pattern" || info.Text == "" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}

func TestDelayedStepPlayerComposition(t *testing.T) {
	player := dsl.BindPlayer("future player")
	var pattern = dsl.TurnStepBegins(dsl.EachUpkeep, player)
	var delayed dsl.DelayedStepPattern = pattern
	body := dsl.AtNext(delayed, dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(2)))
	for _, ability := range []mage.Ability{
		dsl.Spell(body),
		dsl.Activate(dsl.Costs(), body),
		dsl.At(dsl.EachEndStep, body),
		dsl.ActivateModes(dsl.Costs(), dsl.ModeOf("Delay", body), dsl.ModeOf("Life", dsl.GainLife(dsl.You, dsl.Fixed(1)))),
	} {
		card := dsl.NewEnchantment("Schedule", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(body)
	if info.Role != "delayed-step" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v; properties = %+v", info, body.Properties())
	}
}
