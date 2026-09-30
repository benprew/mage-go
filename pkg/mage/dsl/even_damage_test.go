package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestEvenDamagePublicDeclaration(t *testing.T) {
	recipients := dsl.TargetDamageRecipientsAnyNumber("recipients")
	effect := dsl.DealDamageEvenly(dsl.This, recipients, dsl.ValueAtResolution(dsl.XValue()))
	card := dsl.NewSorcery("Synthetic Equal Division", "{X}{R}", dsl.Spell(effect, dsl.WithTargets(recipients)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "resolving-effect" || len(metadata.Children) != 3 || !metadata.Properties.Mass || metadata.Properties.Outcome != dsl.OutcomeDetriment || metadata.Text != "deal damage divided evenly, rounded down" {
		t.Fatal("missing division or damage metadata")
	}
}
