package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAttachedControllerUpkeepComposition(t *testing.T) {
	host, player := dsl.BindPermanent("host"), dsl.BindPlayer("player")
	pattern := dsl.AttachedControllerUpkeep(host, player)
	body := dsl.Sequence(dsl.Tap(dsl.From(host)), dsl.GainLife(dsl.FromPlayer(player), dsl.Fixed(1)))
	for _, ability := range []dsl.TriggeredAbility{
		dsl.WhenEvent(pattern, body),
		dsl.WhenEventModes(pattern, dsl.ModeOf("Now", body), dsl.ModeOf("Later", dsl.AtNext(dsl.EachEndStep, body))),
		dsl.WhenEvent(pattern, dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)),
	} {
		card := dsl.NewAura("Watcher", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	info := dsl.InspectDefinition(pattern)
	if info.Role != "event-pattern" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}
