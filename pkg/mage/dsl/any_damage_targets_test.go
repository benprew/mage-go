package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnyNumberDamageTargetsPublicDeclaration(t *testing.T) {
	recipients := dsl.TargetDamageRecipientsAnyNumber("recipients")
	card := dsl.NewSorcery("Any number damage", "{X}{R}", dsl.Spell(dsl.DealAllocatedDamageX(dsl.This, recipients), dsl.WithTargets(recipients)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := dsl.InspectDefinition(dsl.DealDamageEach(dsl.This, recipients, dsl.Fixed(2)))
	if metadata.Role != "resolving-effect" || !metadata.Properties.Mass || metadata.Properties.Outcome != dsl.OutcomeDetriment {
		t.Fatal("missing damage metadata")
	}
}
