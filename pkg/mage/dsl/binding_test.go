package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSavedPermanentComposition(t *testing.T) {
	saved := dsl.BindPermanent("host")
	var effect = dsl.Sequence(
		dsl.Snapshot(dsl.AttachedPermanent, saved),
		dsl.Sacrifice(dsl.This),
		dsl.Sacrifice(dsl.From(saved)),
	)
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "sequence" || len(metadata.Children) != 3 || metadata.Children[0].Role != "snapshot" {
		t.Fatalf("metadata = %+v", metadata)
	}
}

func TestAttachedActivationComposition(t *testing.T) {
	card := dsl.NewAura("Attached activation", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.Mana("{G}")), dsl.Regenerate(dsl.AttachedPermanent),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestNumberSnapshotComposition(t *testing.T) {
	saved := dsl.BindNumber("amount")
	body := dsl.Sequence(dsl.SnapshotNumber(dsl.Fixed(3), saved), dsl.GainLife(dsl.You, dsl.FromNumber(saved)))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Modes", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Save", body), dsl.ModeOf("Gain", dsl.GainLife(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPermanent("member"), body)))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	metadata := dsl.InspectDefinition(dsl.SnapshotNumber(dsl.Fixed(3), saved))
	if metadata.Role != "number-snapshot" || len(metadata.Children) != 2 {
		t.Fatalf("metadata = %+v", metadata)
	}
}
