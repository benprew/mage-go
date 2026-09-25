package jumpstart

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestOvergrownBattlementImmediateMana(t *testing.T) {
	for _, otherDefender := range []bool{false, true} {
		for _, changedController := range []bool{false, true} {
			t.Run(fmt.Sprintf("extra_defender=%t/nonowner=%t", otherDefender, changedController), func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				controller := a
				if changedController {
					controller = b
				}
				card, err := mage.CreateCard("Overgrown Battlement")
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, controller.PlayerID())
				source.GrantBaseAttr(Haste)
				g.PutOnBattlefield(mage.NewCreature("Nondefender", "", 1, 1), controller.PlayerID())
				g.PutOnBattlefield(mage.NewArtifact("Noncreature defender", "", mage.WithKeyword(Defender)), controller.PlayerID())
				opponent := b
				if changedController {
					opponent = a
				}
				g.PutOnBattlefield(mage.NewCreature("Opposing defender", "", 0, 4, mage.WithKeyword(Defender)), opponent.PlayerID())
				want := 1
				if otherDefender {
					g.PutOnBattlefield(mage.NewCreature("Controlled defender", "", 0, 4, mage.WithKeyword(Defender)), controller.PlayerID())
					want++
				}
				if err := g.ActivateAbilityByIndex(controller.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.StackSize() != 0 {
					t.Fatal("mana ability used the stack")
				}
				if got := controller.ManaPool().Count(Green); got != want {
					t.Fatalf("green mana = %d, want %d", got, want)
				}
				if opponent.ManaPool().Count(Green) != 0 {
					t.Fatal("mana went to the other player")
				}
				if !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("tap cost was not paid")
				}
			})
		}
	}
}
