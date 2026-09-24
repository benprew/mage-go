package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedTargetOwnerComposition(t *testing.T) {
	target := dsl.TargetPermanent("owned", dsl.AnyPermanent, dsl.TargetYouOwn(), dsl.TargetYouControl())
	group := dsl.TargetPermanents("owned group", 1, 2, dsl.IsCreature, dsl.TargetYouOwn())
	saved := dsl.BindPermanent("member")
	body := dsl.ReturnToHand(target)
	cards := []mage.Card{
		dsl.NewArtifact("Obelisk of Undoing composition", "{1}", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{6}"), dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Owned spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewCreature("Owned trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Owned mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Return", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Owned group", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(group, saved, dsl.ReturnToHand(dsl.From(saved))), dsl.WithTargets(group)))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var owner bool
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			owner = owner || n.Role == "target-owner" && n.Text == "you own"
			for _, child := range n.Children {
				visit(child)
			}
		}
		for _, ability := range card.Abilities() {
			visit(dsl.InspectDefinition(ability))
		}
		if !owner {
			t.Fatalf("%s does not expose ownership metadata", card.Name())
		}
	}
	undeclared := dsl.NewInstant("Undeclared", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition(undeclared.Name(), undeclared); err == nil {
		t.Fatal("accepted undeclared owned target")
	}
}
