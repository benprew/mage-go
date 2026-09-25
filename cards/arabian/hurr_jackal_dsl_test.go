package arabian

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestHurrJackalTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Hurr Jackal")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Hurr Jackal", card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(mage.ActivatedAbility)
	if len(ability.Targets()) != 1 {
		t.Fatal("expected one target")
	}
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed creature target")
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	for _, owner := range []*mage.BasePlayer{a, b} {
		for _, creature := range []bool{false, true} {
			c := mage.NewArtifact("Artifact", "")
			if creature {
				c = mage.NewCreature("Creature", "", 2, 2)
			}
			c.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(c, owner.PlayerID())
			if slices.Contains(ability.Targets()[0].Possible(a.PlayerID(), card, g), c.ID()) != creature {
				t.Fatal("incorrect target eligibility")
			}
		}
	}
}
