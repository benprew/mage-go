package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCastConditionPublicDefinition(t *testing.T) {
	card := dsl.NewInstant("Cast condition test", "", dsl.Spell(
		dsl.IfCast(dsl.ControlledCreatureSubtypeAtCast("Dragon"), dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
