package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPermanentDeathOutcomeDSL(t *testing.T) {
	former := dsl.BindPlayer("former controller")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.PermanentDies().OfType(core.TypeArtifact).YouControlled().NotSacrificed().WithController(former),
		dsl.GainLife(dsl.FromPlayer(former), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
