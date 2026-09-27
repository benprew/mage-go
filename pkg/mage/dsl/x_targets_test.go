package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestXTargetPermanentDeclarations(t *testing.T) {
	lands := dsl.TargetPermanentsExactlyX("lands", dsl.IsLand)
	member := dsl.BindPermanent("land")
	ability := dsl.Activate(dsl.Costs(dsl.Mana("{X}"), dsl.TapCost(dsl.This)),
		dsl.ForEach(lands, member, dsl.Untap(dsl.From(member))),
		dsl.WithTargets(lands))
	card := dsl.NewArtifact("Example", "", dsl.WithAbility(ability))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}

	creatures := dsl.TargetPermanentsUpToX("creatures", dsl.IsCreature)
	spell := dsl.NewInstant("Example spell", "{X}", dsl.WithAbility(dsl.Spell(
		dsl.DestroyEach(creatures), dsl.WithTargets(creatures))))
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
}
