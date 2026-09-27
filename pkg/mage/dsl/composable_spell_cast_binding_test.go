package dsl

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestSpellCastBindingPublicDeclaration(t *testing.T) {
	spell := BindSpell("cast spell")
	card := NewArtifact("Cast Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell), Counter(FromSpell(spell)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
