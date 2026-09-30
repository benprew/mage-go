package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAllocatedDamageXPublicDeclaration(t *testing.T) {
	recipients := dsl.TargetDamageRecipientsUpToX("recipients")
	effect := dsl.DealAllocatedDamageX(dsl.This, recipients)
	card := dsl.NewSorcery("Allocated X", "{X}{R}", dsl.Spell(dsl.Sequence(effect, dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(recipients)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "resolving-effect" || len(metadata.Children) != 3 || metadata.Children[2].Role != "announcement-value" || metadata.Children[2].Text != "announced X" || metadata.Properties.DamageValue.Text() != "X" || !metadata.Properties.Mass {
		t.Fatal("missing announcement or damage metadata")
	}
}
