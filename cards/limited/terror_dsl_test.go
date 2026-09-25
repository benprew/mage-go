package limited

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTerrorTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Terror")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Terror", card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
	if len(ability.Targets()) != 1 {
		t.Fatal("expected one target")
	}
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed creature target")
	}
	info := mage.InspectDefinition(ability.Effects()[0])
	if info.Text != "destroy without regeneration creature" || !info.Properties.Destroys {
		t.Fatalf("wrong destruction definition: %+v", info)
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	for _, owner := range []*mage.BasePlayer{a, b} {
		for _, kind := range []string{"green", "black", "multicolor", "artifact", "land"} {
			c := mage.NewCreature(kind, "{G}", 2, 2)
			switch kind {
			case "black":
				c = mage.NewCreature(kind, "{B}", 2, 2)
			case "multicolor":
				c = mage.NewCreature(kind, "{B}{G}", 2, 2)
			case "artifact":
				c.AddType(core.TypeArtifact)
			case "land":
				c = mage.NewLand(kind)
			}
			c.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(c, owner.PlayerID())
			got := ability.Targets()[0].Possible(a.PlayerID(), card, g)
			if slices.Contains(got, c.ID()) != (kind == "green") {
				t.Fatalf("incorrect target eligibility for %s, types %v", kind, c.HasType(core.TypeCreature))
			}
		}
	}
}
