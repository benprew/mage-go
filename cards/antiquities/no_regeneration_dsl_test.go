package antiquities

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestShatterstormTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Shatterstorm")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
	info := mage.InspectDefinition(ability.Effects()[0])
	if len(ability.Targets()) != 0 || info.Role != "destroy-each-without-regeneration" || len(info.Children) != 1 || info.Children[0].Text != mage.IsArtifact.Text() {
		t.Fatalf("wrong collection: %+v", info)
	}
	if !info.Properties.Destroys || !info.Properties.Mass || info.Properties.Outcome != mage.OutcomeDetriment {
		t.Fatal("missing destruction metadata")
	}
}
