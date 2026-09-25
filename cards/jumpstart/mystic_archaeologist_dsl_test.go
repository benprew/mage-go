package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestMysticArchaeologistDrawPipeline(t *testing.T) {
	for _, scenario := range []string{"present", "departed", "short library", "replaced empty draws"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			wantDraws := 2
			switch scenario {
			case "short library":
				wantDraws = 1
			case "replaced empty draws":
				wantDraws = 0
			}
			for range wantDraws {
				controller.AddToLibrary(mage.NewArtifact("Drawn card", ""))
			}
			drawEvents := 0
			watcher := mage.NewTriggered(core.EvtCardDrawn, false, mage.DrawFor(mage.You, mage.Fixed(1))).
				SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
					drawEvents++
					if evt.PlayerID != controller.PlayerID() {
						t.Error("draw event has the wrong player")
					}
					return false
				})
			watcherCard := mage.NewArtifact("Watcher", "", mage.WithAbility(watcher))
			g.PutOnBattlefield(watcherCard, owner.PlayerID())
			replacements := 0
			if scenario == "replaced empty draws" {
				g.AddEmptyLibraryDrawReplacement(watcherCard.ID(), controller.PlayerID(), func(_ *mage.Game, _ uuid.UUID) { replacements++ })
			}
			card, err := mage.CreateCard("Mystic Archaeologist")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(owner.PlayerID())
			g.PutOnBattlefield(card, controller.PlayerID())
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation must fail without changing the stack", err)
			}
			controller.ManaPool().Add(core.Blue, 2)
			controller.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || controller.ManaPool().TotalMana() != 0 {
				t.Fatal("activation must pay five mana and use the stack")
			}
			if len(controller.Hand()) != 0 || drawEvents != 0 || replacements != 0 {
				t.Fatal("activation drew cards before resolution")
			}
			if len(g.StackPeek().Targets) != 0 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("activation must have no targets")
			}
			if scenario == "departed" {
				g.ExilePermanent(g.FindPermanent(card.ID()))
			}
			g.ResolveStack()
			if len(controller.Hand()) != wantDraws || drawEvents != wantDraws {
				t.Errorf("hand/draw events = %d/%d, want %d/%d", len(controller.Hand()), drawEvents, wantDraws, wantDraws)
			}
			if len(owner.Hand()) != 0 {
				t.Error("activation drew for the owner instead of the controller")
			}
			if scenario == "replaced empty draws" && replacements != 2 {
				t.Errorf("replaced draws = %d, want 2", replacements)
			}
			if controller.IsAlive() != (scenario != "short library") {
				t.Error("incorrect empty-library draw outcome")
			}
		})
	}
}
