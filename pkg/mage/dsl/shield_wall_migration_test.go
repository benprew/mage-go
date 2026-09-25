package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestShieldWallTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Shield Wall")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Shield Wall", card); err != nil {
		t.Fatal(err)
	}
	spell, ok := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
	if !ok {
		t.Fatalf("ability = %T, want spell", card.Abilities()[0])
	}
	if len(spell.Targets()) != 0 {
		t.Fatal("collection spell must not target")
	}
	info := mage.InspectDefinition(spell.Effects()[0])
	if info.Role != "until" {
		t.Fatalf("effect role = %q, want until", info.Role)
	}
	props := info.Properties
	if !props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != 0 || props.ToughnessBoost != 2 {
		t.Fatalf("boost metadata = %+v", props)
	}
}
