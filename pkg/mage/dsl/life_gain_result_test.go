package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestLifeGainedIntoPublicDeclaration(t *testing.T) {
	gained := dsl.BindNumber("life gained")
	card := dsl.NewArtifact("Synthetic", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Sequence(
		dsl.GainLifeEach(dsl.EachPlayer, dsl.Fixed(2), dsl.LifeGainedInto(gained)),
		dsl.LoseLife(dsl.You, dsl.FromNumber(gained)),
	))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
