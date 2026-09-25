package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestClassicCardEffectsAreReusable(t *testing.T) {
	tests := []struct {
		name   string
		effect Effect
	}{
		{"Berserk", BerserkEffect()},
		{"Lightning Bolt", LightningBoltEffect()},
		{"Prodigal Sorcerer", ProdigalSorcererEffect()},
		{"Sorceress Queen", SorceressQueenEffect()},
		{"Swords to Plowshares", SwordsToPlowsharesEffect()},
		{"Tawnos's Wand", TawnosWandEffect()},
		{"Lifelace", LaceEffect(Green)},
		{"Lesser Werewolf", LesserWerewolfCounterEffect()},
		{"Aladdin's Ring", AladdinsRingEffect()},
		{"Bottle of Suleiman", BottleOfSuleimanEffect()},
		{"Healing Salve prevention", HealingSalvePreventionEffect()},
		{"Nevinyrral's Disk", DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment)))},
		{"Pandora's Box", PandorasBoxEffect()},
		{"Sindbad", SindbadEffect()},
		{"The Hive", TheHiveEffect()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.effect == nil {
				t.Fatal("effect is nil")
			}
			if tt.effect.Text() == "" {
				t.Fatal("effect has no text")
			}
		})
	}
}
