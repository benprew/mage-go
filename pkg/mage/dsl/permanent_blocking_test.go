package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentBlockingPublicDefinition(t *testing.T) {
	card := dsl.NewAura("Blocking cloak", "",
		dsl.WithAbility(dsl.Static(dsl.While(
			dsl.PermanentIsBlocking(dsl.AttachedPermanent),
			dsl.GrantKeyword(dsl.AttachedPermanent, dsl.Shroud),
		))),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
