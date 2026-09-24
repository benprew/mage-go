package limited

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestLandDestructionTypedDefinitions(t *testing.T) {
	for _, name := range []string{"Sinkhole", "Stone Rain", "Ice Storm"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			if len(ability.Targets()) != 1 {
				t.Fatal("expected one target")
			}
			_, ok := ability.Targets()[0].(mage.TargetBinding)
			if !ok {
				t.Fatal("missing typed land target")
			}
			properties := ability.Effects()[0].Properties()
			if !properties.Destroys || properties.Outcome != mage.OutcomeDetriment {
				t.Fatal("missing destruction metadata")
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			lands := make([]uuid.UUID, 0, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				land := mage.NewLand("Land")
				land.SetOwner(player.PlayerID())
				g.PutOnBattlefield(land, player.PlayerID())
				lands = append(lands, land.ID())
				creature := mage.NewCreature("Creature", "", 2, 2)
				creature.SetOwner(player.PlayerID())
				g.PutOnBattlefield(creature, player.PlayerID())
			}
			got := ability.Targets()[0].Possible(a.PlayerID(), card, g)
			if len(got) != 2 || !slices.Contains(got, lands[0]) || !slices.Contains(got, lands[1]) {
				t.Fatalf("land targets = %v, want %v", got, lands)
			}
		})
	}
}
