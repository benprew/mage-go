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
		{"Blood Lust", BloodLustEffect()},
		{"Flying Carpet", FlyingCarpetEffect()},
		{"Giant Growth", GiantGrowthEffect()},
		{"Helm of Chatzuk", HelmOfChatzukEffect()},
		{"Hurr Jackal", HurrJackalEffect()},
		{"Lightning Bolt", LightningBoltEffect()},
		{"Prodigal Sorcerer", ProdigalSorcererEffect()},
		{"Sorceress Queen", SorceressQueenEffect()},
		{"Staff of Zegon", StaffOfZegonEffect()},
		{"Swords to Plowshares", SwordsToPlowsharesEffect()},
		{"Tawnos's Wand", TawnosWandEffect()},
		{"Twiddle", TwiddleEffect()},
		{"Unsummon", UnsummonEffect()},
		{"Lifelace", LaceEffect(Green)},
		{"Lesser Werewolf", LesserWerewolfCounterEffect()},
		{"Aladdin's Ring", AladdinsRingEffect()},
		{"Ancestral Recall", AncestralRecallEffect()},
		{"Boomerang", BoomerangEffect()},
		{"Bottle of Suleiman", BottleOfSuleimanEffect()},
		{"Crumble", CrumbleEffect()},
		{"Disenchant", DisenchantEffect()},
		{"Disrupting Scepter", DisruptingScepterEffect()},
		{"Fissure", FissureEffect()},
		{"Fog", FogEffect()},
		{"Healing Salve life", HealingSalveGainEffect()},
		{"Healing Salve prevention", HealingSalvePreventionEffect()},
		{"Millstone", MillstoneEffect()},
		{"Nevinyrral's Disk", NevinyrralsDiskEffect()},
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
