package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlightCapturesTappedLand(t *testing.T) {
	for _, change := range []string{"move", "leave", "return land", "regenerate", "indestructible", "none"} {
		t.Run(change, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			aura, err := mage.CreateCard("Blight")
			if err != nil {
				t.Fatal(err)
			}
			first, second := mage.NewLand("First"), mage.NewLand("Second")
			for _, card := range []mage.Card{aura, first, second} {
				card.SetOwner(a.PlayerID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			g.Attach(aura.ID(), first.ID())
			g.TapPermanent(g.FindPermanent(first.ID()))
			switch change {
			case "regenerate":
				g.AddRegenerationShield(first.ID())
			case "indestructible":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Indestructible)
			case "move":
				g.Attach(aura.ID(), second.ID())
			case "leave":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
			case "return land":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, a.PlayerID())
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("expected one nontargeted trigger")
			}
			g.ResolveStack()
			if (g.FindPermanent(first.ID()) != nil) != (change == "return land" || change == "regenerate" || change == "indestructible") {
				t.Fatal("wrong land incarnation destroyed")
			}
			if g.FindPermanent(second.ID()) == nil {
				t.Fatal("new attachment destroyed")
			}
		})
	}
}
