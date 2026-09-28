package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastCreatureTargetPublicDeclaration(t *testing.T) {
	card := dsl.NewArtifact("Repartee source", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(),
		dsl.Draw(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
