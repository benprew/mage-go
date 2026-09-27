package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastEnchantmentSubtypeDSL(t *testing.T) {
	pattern := dsl.SpellCast().ByOpponent().OfType(core.TypeEnchantment).OfEnchantmentSubtype("Aura")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern, dsl.Draw(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("inspection=%+v", got)
	}
}
