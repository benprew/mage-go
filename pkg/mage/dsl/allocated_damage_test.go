package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAllocatedDamagePublicDeclaration(t *testing.T) {
	recipients := dsl.TargetDamageRecipients("recipients", 1, 3)
	effect := dsl.DealAllocatedDamage(dsl.This, recipients, 3)
	card := dsl.NewSorcery("Allocated damage", "{2}{R}", dsl.Spell(effect, dsl.WithTargets(recipients)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "resolving-effect" || len(metadata.Children) != 3 || metadata.Properties.DamageValue == nil || !metadata.Properties.Mass {
		t.Fatal("inspection omitted damage declaration or AI metadata")
	}
}

func TestAllocatedDamageSequencePublicDeclaration(t *testing.T) {
	recipients := dsl.TargetDamageRecipients("recipients", 1, 3)
	card := dsl.NewSorcery("Allocated sequence", "{2}{R}",
		dsl.Spell(dsl.Sequence(
			dsl.DealAllocatedDamage(dsl.This, recipients, 3),
			dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(recipients)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
