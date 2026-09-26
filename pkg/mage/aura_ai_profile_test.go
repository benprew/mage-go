package mage

import (
	"testing"
)

func TestAuraAIProfile_DerivedFromComposableDSL(t *testing.T) {
	card := NewAura("Holy Strength", "{W}",
		WithAbility(Static(While(Controls(AnyPermanent), ModifyPT(AttachedPermanent, Fixed(1), Fixed(2))))),
	)

	profile, ok := card.AuraAIProfile()
	if !ok {
		t.Fatal("expected AuraAIProfile to be derived from composable static ability")
	}
	if profile.PowerBoost != 1 || profile.ToughnessBoost != 2 {
		t.Fatalf("expected +1/+2, got +%d/+%d", profile.PowerBoost, profile.ToughnessBoost)
	}
	if profile.StealsControl {
		t.Fatal("expected StealsControl to be false")
	}
}

func TestAuraAIProfile_FallbackToExplicitProfile(t *testing.T) {
	card := NewAura("Control Magic", "{2}{U}{U}",
		WithAuraAIProfile(AuraAIProfile{StealsControl: true}),
	)

	profile, ok := card.AuraAIProfile()
	if !ok {
		t.Fatal("expected AuraAIProfile to fall back to explicit profile")
	}
	if !profile.StealsControl {
		t.Fatal("expected StealsControl to be true")
	}
}

func TestAuraAIProfile_NonAuraReturnsFalse(t *testing.T) {
	card := NewCreature("Bear", "{1}{G}", 2, 2)
	_, ok := card.AuraAIProfile()
	if ok {
		t.Fatal("expected AuraAIProfile to return false for non-aura card")
	}
}
