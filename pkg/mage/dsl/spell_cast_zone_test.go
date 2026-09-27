package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSpellCastZonePublicDefinition(t *testing.T) {
	caster := dsl.BindPlayer("caster")
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
		dsl.SpellCast().FromZone(core.ZoneHand).WithCaster(caster),
		dsl.GainLife(dsl.FromPlayer(caster), dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
