package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedDamageEventPlayer(t *testing.T) {
	player := dsl.BindPlayer("damaged player")
	card := dsl.NewCreature("Source", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.DealsDamageToPlayer(dsl.This, player), dsl.DiscardAtRandom(dsl.FromPlayer(player), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if mage.InspectDefinition(card.Abilities()[0]).Role != "event-trigger" {
		t.Fatal("missing event metadata")
	}
}

func TestTypedDamageEventOpponent(t *testing.T) {
	for _, modal := range []bool{false, true} {
		player := dsl.BindPlayer("damaged opponent")
		pattern := dsl.DealsDamageToOpponent(dsl.This, player)
		body := dsl.DiscardAtRandom(dsl.FromPlayer(player), dsl.Fixed(1))
		trigger := dsl.WhenEvent(pattern, body)
		if modal {
			trigger = dsl.WhenEventModes(pattern, dsl.ModeOf("Discard", body), dsl.ModeOf("Draw", dsl.Draw(dsl.FromPlayer(player), dsl.Fixed(1))))
		}
		card := dsl.NewCreature("Source", "", 1, 1, dsl.WithAbility(trigger))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var opponent, discard bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			opponent = opponent || n.Role == "event-pattern" && n.Text == "source deals damage to an opponent"
			discard = discard || n.Text == "discard at random for damaged opponent" && n.Properties.Outcome == mage.OutcomeDetriment
			for _, child := range n.Children {
				visit(child)
			}
		}
		visit(mage.InspectDefinition(card.Abilities()[0]))
		if !opponent || !discard {
			t.Fatalf("modal=%v opponent=%v discard=%v", modal, opponent, discard)
		}
	}
}
