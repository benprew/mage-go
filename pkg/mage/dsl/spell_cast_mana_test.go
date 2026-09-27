package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastManaSpentPublicDeclaration(t *testing.T) {
	spent := dsl.BindNumber("mana spent")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().WithManaSpent(spent),
		dsl.GainLife(dsl.You, dsl.FromNumber(spent)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
