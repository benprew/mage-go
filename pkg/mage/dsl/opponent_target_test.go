package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedOpponentTargetComposition(t *testing.T) {
	var option = dsl.TargetOpponentOnly()
	target := dsl.TargetPlayer("opponent", option)
	body := dsl.Sequence(dsl.LoseLife(target, dsl.Fixed(1)), dsl.GainLife(dsl.You, dsl.Fixed(1)))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.CreatureDies(), body, dsl.WithTargets(target)))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Drain", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		found := false
		var visit func(mage.DefinitionMetadata)
		visit = func(n mage.DefinitionMetadata) {
			found = found || n.Role == "target-player" && n.Text == "opponent"
			for _, child := range n.Children {
				visit(child)
			}
		}
		for _, ability := range card.Abilities() {
			visit(dsl.InspectDefinition(ability))
		}
		if !found {
			t.Fatalf("%s lacks opponent restriction metadata", card.Name())
		}
	}
	card := dsl.NewInstant("Undeclared", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted undeclared opponent target")
	}
}

func TestTypedOpponentTargetNilOption(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("accepted nil player target option")
		}
	}()
	dsl.TargetPlayer("opponent", nil)
}
