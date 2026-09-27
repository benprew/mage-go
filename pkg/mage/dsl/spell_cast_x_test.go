package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastXPublicDeclaration(t *testing.T) {
	card := NewCreature("X watcher", "{2}{U}", 2, 2, WithAbility(WhenEvent(
		SpellCast().WithXInManaCost(), Until(EndOfTurn, CantBeBlocked(This)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
