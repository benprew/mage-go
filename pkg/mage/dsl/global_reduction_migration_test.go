package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestGlobalReductionTypedDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
	}{{"Hell Swarm", -1, 0}, {"Marsh Gas", -2, 0}, {"Bone Flute", -1, 0}, {"Languish", -4, -4}} {
		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0])
			var effects []mage.Effect
			switch a := ability.(type) {
			case *mage.SpellAbility:
				if len(a.Targets()) != 0 {
					t.Fatal("collection spell must not target")
				}
				effects = a.Effects()
			case mage.ActivatedAbility:
				if a.SorcerySpeed() || len(a.Targets()) != 0 {
					t.Fatal("expected no targets and instant timing")
				}
				effects = a.Effects()
			default:
				t.Fatalf("unexpected ability %T", ability)
			}
			info := mage.InspectDefinition(effects[0])
			if info.Role != "until" {
				t.Fatalf("effect role = %q, want until", info.Role)
			}
			props := info.Properties
			if !props.Mass || props.Outcome != mage.OutcomeDetriment || props.PowerBoost != tc.power || props.ToughnessBoost != tc.toughness {
				t.Fatalf("reduction metadata = %+v", props)
			}
		})
	}
}
