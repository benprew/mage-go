package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSingularLifeGainResultPublicDeclaration(t *testing.T) {
	gained := dsl.BindNumber("gained")
	card := dsl.NewSorcery("Life result", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
		dsl.GainLifeForWithResult(dsl.You, dsl.Fixed(2), dsl.LifeGainedInto(gained)),
		dsl.LoseLife(dsl.You, dsl.FromNumber(gained)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
