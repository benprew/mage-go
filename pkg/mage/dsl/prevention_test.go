package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedPreventionAmountComposition(t *testing.T) {
	saved := dsl.BindPermanent("amount source")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.PreventNextDamage(dsl.You, dsl.MaxValue(dsl.Fixed(0), dsl.PowerOf(dsl.From(saved)))))
	recipient := dsl.TargetDamageRecipient("recipient")
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, card := range []mage.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.PreventNextDamage(recipient, dsl.ToughnessOf(target)), dsl.WithTargets(recipient, target)))),
		dsl.NewCreature("Activation", "", 2, 3, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
		dsl.NewCreature("Trigger", "", 2, 3, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Shield", dsl.PreventNextDamage(dsl.You, dsl.Fixed(2))), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
