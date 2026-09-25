package jumpstart

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRunedServitorDeathDrawOrder(t *testing.T) {
	for _, scenario := range []string{"active owner", "active controller", "source returns"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			g.SetActivePlayerIndex(0)
			wantOrder := []uuid.UUID{a.PlayerID(), b.PlayerID()}
			if scenario != "active owner" {
				g.SetActivePlayerIndex(1)
				wantOrder = []uuid.UUID{b.PlayerID(), a.PlayerID()}
			}
			a.AddToLibrary(mage.NewArtifact("A card", ""))
			b.AddToLibrary(mage.NewArtifact("B card", ""))
			var order []uuid.UUID
			watcher := mage.NewTriggered(core.EvtCardDrawn, false, mage.DrawFor(mage.You, mage.Fixed(1))).
				SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
					order = append(order, evt.PlayerID)
					return false
				})
			g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), a.PlayerID())
			card, err := mage.CreateCard("Runed Servitor")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, b.PlayerID())
			g.DestroyPermanent(g.FindPermanent(card.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
				t.Fatal("expected one death trigger for the last controller")
			}
			if len(g.StackPeek().Targets) != 0 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Error("death trigger must have no targets")
			}
			if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
				t.Fatal("draw occurred before resolution")
			}
			if scenario == "source returns" {
				returned, ok := a.RemoveFromGraveyard(card.ID())
				if !ok {
					t.Fatal("source is absent from its owner's graveyard")
				}
				g.PutOnBattlefield(returned, a.PlayerID())
			}
			g.ResolveStack()
			g.AssertHandCount(gametest.PlayerA, "A card", 1)
			g.AssertHandCount(gametest.PlayerB, "B card", 1)
			if !slices.Equal(order, wantOrder) {
				t.Errorf("draw event order = %v, want %v", order, wantOrder)
			}
		})
	}
}
