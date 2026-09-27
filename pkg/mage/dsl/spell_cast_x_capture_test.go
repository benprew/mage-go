package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastAnnouncedXPublicDeclaration(t *testing.T) {
	x := dsl.BindNumber("cast X")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().WithXInManaCost().WithAnnouncedX(x),
		dsl.GainLife(dsl.You, dsl.FromNumber(x)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
