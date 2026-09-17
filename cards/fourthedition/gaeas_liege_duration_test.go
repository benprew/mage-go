package fourthedition

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGaeasLiegeSourceDuration(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased", "target returned"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.PutOnBattlefield(mage.NewLand("Forest", mage.WithSubTypes("Forest")), a.PlayerID())
			card, err := mage.CreateCard("Gaea's Liege")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			landCard := mage.NewLand("Mountain", mage.WithSubTypes("Mountain"), mage.WithManaAbility(core.Red))
			landCard.SetOwner(b.PlayerID())
			land := g.PutOnBattlefield(landCard, b.PlayerID())
			abilityIndex := -1
			for i, ability := range source.RuntimeAbilities {
				if _, ok := ability.(mage.ActivatedAbility); ok {
					abilityIndex = i
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), abilityIndex, []uuid.UUID{land.ID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "phased":
				g.PhaseOut(source.ID())
			case "target returned":
				g.RemoveFromBattlefield(land)
				g.PutOnBattlefield(landCard, b.PlayerID())
			default:
				g.RemoveFromBattlefield(source)
				if state == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			g.ResolveStack()
			land = g.FindPermanent(landCard.ID())
			if got, want := land.HasSubType("Forest"), state == "phased"; got != want {
				t.Fatalf("land is Forest = %v, want %v", got, want)
			}
		})
	}
}
