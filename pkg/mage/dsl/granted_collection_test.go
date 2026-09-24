package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestGrantedCollectionComposition(t *testing.T) {
	target := dsl.TargetPermanent("target", dsl.IsCreature)
	set := dsl.ControlledPermanents(dsl.IsArtifact)
	for _, mod := range []dsl.ContinuousModification{
		dsl.EachActivatedAbilityModification(set, dsl.Costs(), dsl.Regenerate(target), dsl.WithTargets(target), dsl.ActivationLimit(1)),
		dsl.EachActivatedModesModification(set, dsl.Costs(), dsl.ModeOf("regenerate", dsl.Regenerate(target), dsl.WithTargets(target)), dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.OnlyAsSorcery()),
	} {
		card := dsl.NewArtifact("Grant", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Until(core.EndOfTurn, mod))), dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsArtifact), mod))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		effect := dsl.Until(core.EndOfTurn, mod)
		if !effect.Properties().Mass || effect.Properties().Outcome != mage.OutcomeBenefit {
			t.Fatal("missing mass benefit metadata")
		}
	}
	outer := dsl.BindPermanent("outer")
	invalid := dsl.NewInstant("Invalid", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.Snapshot(dsl.This, outer), dsl.Until(core.EndOfTurn, dsl.EachActivatedAbilityModification(set, dsl.Costs(), dsl.Regenerate(dsl.From(outer))))))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("grant read outer binding")
	}
}
