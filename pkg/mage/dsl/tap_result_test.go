package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedTapResultDeclaration(t *testing.T) {
	done := BindNumber("tap result")
	card := NewArtifact("Synthetic tap", "", WithAbility(Activate(Costs(), Sequence(
		Tap(This, TapAmountInto(done)),
		IfPositive(FromNumber(done), GainLife(You, Fixed(2))),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
