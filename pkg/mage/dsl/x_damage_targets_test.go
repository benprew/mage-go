package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestXDamageTargetsPublicDeclaration(t *testing.T) {
	for _, targets := range []*dsl.DamageTargetsBinding{dsl.TargetDamageRecipientsExactlyX("recipients"), dsl.TargetDamageRecipientsUpToX("recipients")} {
		card := dsl.NewInstant("X recipients", "{X}{R}", dsl.Spell(dsl.DealDamageEach(dsl.This, targets, dsl.Fixed(2)), dsl.WithTargets(targets)))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		metadata := dsl.InspectDefinition(dsl.DealDamageEach(dsl.This, targets, dsl.Fixed(2)))
		if metadata.Role != "resolving-effect" || len(metadata.Children) != 3 || !metadata.Properties.Mass {
			t.Fatal("missing damage metadata")
		}
	}
}
