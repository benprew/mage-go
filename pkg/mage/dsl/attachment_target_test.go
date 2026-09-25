package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAttachmentTargetComposition(t *testing.T) {
	target := dsl.TargetPermanent("aura", dsl.And(dsl.IsEnchantment, dsl.HasSubType("Aura"), dsl.IsAttachedTo(dsl.IsCreature)), dsl.TargetAttachedToYouControl())
	ability := dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), dsl.Destroy(target), dsl.WithTargets(target))
	card := dsl.NewCreature("Worker", "{W}", 1, 1, dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	found := false
	var visit func(mage.DefinitionMetadata)
	visit = func(n mage.DefinitionMetadata) {
		found = found || n.Role == "target-attachment-controller" && n.Text == "you control"
		for _, child := range n.Children {
			visit(child)
		}
	}
	visit(dsl.InspectDefinition(ability))
	if !found {
		t.Fatal("attachment controller metadata is absent")
	}
	invalid := dsl.NewInstant("Undeclared", "", dsl.WithAbility(dsl.Spell(dsl.Destroy(target))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("undeclared target was accepted")
	}
}
