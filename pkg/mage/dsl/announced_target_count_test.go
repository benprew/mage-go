package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnnouncedTargetCountDeclaration(t *testing.T) {
	targets := dsl.TargetPermanentsUpToX("creatures", dsl.IsCreature)
	card := dsl.NewInstant("Test", "{X}", dsl.WithAbility(dsl.Spell(
		dsl.GainLife(dsl.You, dsl.AnnouncedTargetCount(targets)), dsl.WithTargets(targets))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
