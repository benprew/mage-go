package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestOneTokenResultPublicDeclaration(t *testing.T) {
	created := dsl.BindPermanent("created Cat")
	cat := dsl.DefineToken("Cat", dsl.TokenTypes(core.TypeCreature), dsl.TokenSubtypes("Cat"), dsl.TokenColors(core.White), dsl.TokenPT(2, 2))
	card := dsl.NewInstant("Cat Maker", "", dsl.Spell(dsl.Sequence(
		dsl.CreateOneToken(dsl.You, cat, created),
		dsl.AddCounters(dsl.From(created), core.P1P1, dsl.Fixed(1)),
	)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
