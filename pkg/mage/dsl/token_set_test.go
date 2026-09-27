package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestCreatedTokenSetComposition(t *testing.T) {
	created := dsl.BindCreatedTokens("created")
	member := dsl.BindPermanent("member")
	cat := dsl.DefineToken("Cat", dsl.TokenTypes(core.TypeCreature), dsl.TokenPT(1, 1))
	body := dsl.Sequence(
		dsl.CreateTokenCopiesInto(dsl.You, dsl.Fixed(2), cat, created),
		dsl.ForEach(dsl.FromCreatedTokens(created), member, dsl.Tap(dsl.From(member))),
	)
	card := dsl.NewInstant("Maker", "", dsl.WithAbility(dsl.Spell(body)))
	if err := mage.ValidateCardDefinition("Maker", card); err != nil {
		t.Fatal(err)
	}
}
