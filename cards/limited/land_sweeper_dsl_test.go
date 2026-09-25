package limited

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestLandSweepersTypedDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		filter mage.PermanentFilter
	}{
		{"Armageddon", mage.IsLand},
		{"Wrath of God", mage.IsCreature},
		{"Tsunami", mage.HasSubType("Island")},
		{"Flashfires", mage.HasSubType("Plains")},
		{"Tranquility", mage.IsEnchantment},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			if len(ability.Targets()) != 0 || len(ability.Effects()) != 1 {
				t.Fatal("expected one effect without targets")
			}
			role := "destroy-each"
			if tc.name == "Wrath of God" {
				role = "destroy-each-without-regeneration"
			}
			info := mage.InspectDefinition(ability.Effects()[0])
			if info.Role != role || len(info.Children) != 1 || info.Children[0].Role != "permanent-set" || info.Children[0].Text != tc.filter.Text() {
				t.Fatalf("missing typed destruction collection: %+v", info)
			}
			if !info.Properties.Destroys || !info.Properties.Mass || info.Properties.Outcome != mage.OutcomeDetriment {
				t.Fatal("missing mass destruction metadata")
			}
		})
	}
}
