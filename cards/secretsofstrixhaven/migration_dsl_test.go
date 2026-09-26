package secretsofstrixhaven

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestComposableMigrationTargets(t *testing.T) {
	for _, name := range []string{"Chelonian Tackle", "Cost of Brilliance", "Fractalize", "Growth Curve", "Harsh Annotation", "Homesickness", "Oracle's Restoration"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, ability := range card.Abilities() {
				spell, ok := mage.UnwrapAbility(ability).(*mage.SpellAbility)
				if !ok {
					continue
				}
				found = true
				if len(spell.Targets()) == 0 {
					t.Fatal("spell has no targets")
				}
				for _, target := range spell.Targets() {
					if _, ok := target.(mage.TargetBinding); !ok {
						t.Fatal("spell requires typed targets")
					}
				}
			}
			if !found {
				t.Fatal("card has no spell ability")
			}
		})
	}
}
