package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestEntryDrawResolution(t *testing.T) {
	for _, tc := range []struct {
		name     string
		draws    int
		lifeLoss int
	}{
		{"Wall of Blossoms", 1, 0},
		{"Phyrexian Rager", 1, 1},
		{"Tithebearer Giant", 1, 1},
		{"Phyrexian Gargantua", 2, 2},
	} {
		name := tc.name
		for _, scenario := range []string{"present", "departed", "empty library", "one-card library"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				owner := g.GetPlayer(gametest.PlayerA)
				controller := g.GetPlayer(gametest.PlayerB)
				wantDraws := tc.draws
				switch scenario {
				case "empty library":
					wantDraws = 0
				case "one-card library":
					wantDraws = 1
				}
				for range wantDraws {
					controller.AddToLibrary(mage.NewArtifact("Drawn card", ""))
				}
				drawEvents := 0
				watcher := mage.NewTriggered(core.EvtCardDrawn, false, mage.DrawFor(mage.You, mage.Fixed(1))).
					SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
						drawEvents++
						if evt.PlayerID != controller.PlayerID() || controller.Life() != 20 {
							t.Error("draw must occur for the trigger controller before life loss")
						}
						return false
					})
				g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), owner.PlayerID())
				card, err := mage.CreateCard(name)
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(owner.PlayerID())
				g.PutOnBattlefield(card, controller.PlayerID())
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("expected one entry trigger")
				}
				trigger := g.StackPeek()
				if trigger.Controller != controller.PlayerID() {
					t.Fatal("entry trigger has the wrong controller")
				}
				if len(trigger.Targets) != 0 || len(trigger.TargetSpecs) != 0 {
					t.Error("entry trigger must have no targets")
				}
				if len(controller.Hand()) != 0 || controller.Life() != 20 {
					t.Fatal("entry ability acted before resolution")
				}
				if scenario == "departed" {
					g.ExilePermanent(g.FindPermanent(card.ID()))
				}
				g.ResolveStack()
				wantLife := 20 - tc.lifeLoss
				if len(controller.Hand()) != wantDraws || drawEvents != wantDraws || controller.Life() != wantLife {
					t.Errorf("hand/draw events/life = %d/%d/%d, want %d/%d/%d", len(controller.Hand()), drawEvents, controller.Life(), wantDraws, wantDraws, wantLife)
				}
				if len(owner.Hand()) != 0 || owner.Life() != 20 {
					t.Error("entry ability affected the owner instead of the controller")
				}
				if controller.IsAlive() != (wantDraws == tc.draws) {
					t.Error("incorrect empty-library draw outcome")
				}
			})
		}
	}
}
