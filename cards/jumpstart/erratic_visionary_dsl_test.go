package jumpstart

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestErraticVisionaryDrawDiscardPipeline(t *testing.T) {
	for _, scenario := range []string{"normal", "source leaves", "discard replacement"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			drawn := mage.NewArtifact("Drawn card", "")
			drawn.SetOwner(controller.PlayerID())
			controller.AddToLibrary(drawn)
			var events []core.EventType
			for _, eventType := range []core.EventType{core.EvtCardDrawn, core.EvtDiscard} {
				watcher := mage.NewTriggered(eventType, false, mage.DrawFor(mage.You, mage.Fixed(1))).
					SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
						events = append(events, evt.Type)
						if evt.PlayerID != controller.PlayerID() {
							t.Error("event has the wrong player")
						}
						if evt.Type == core.EvtDiscard && evt.SourceID != drawn.ID() {
							t.Error("discard event has the wrong card")
						}
						return false
					})
				g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), owner.PlayerID())
			}
			if scenario == "discard replacement" {
				g.PutOnBattlefield(mage.NewArtifact("Replacement", "",
					mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), controller.PlayerID())
			}
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Erratic Visionary")
			g.FindPermanent(source).Card.SetOwner(owner.PlayerID())
			controller.ManaPool().Add(core.Blue, 1)
			controller.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), source, 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source).Tapped || controller.ManaPool().TotalMana() != 0 {
				t.Fatal("activation did not pay its mana and tap costs")
			}
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || len(events) != 0 || len(controller.Library()) != 1 {
				t.Fatal("activation must use the stack without targets or early draws")
			}
			if scenario == "source leaves" {
				g.ExilePermanent(g.FindPermanent(source))
			}
			g.ResolveStack()
			if !slices.Equal(events, []core.EventType{core.EvtCardDrawn, core.EvtDiscard}) {
				t.Errorf("events = %v, want draw then discard", events)
			}
			if len(controller.Hand()) != 0 || len(owner.Hand()) != 0 || len(owner.Graveyard()) != 0 {
				t.Error("draw and discard must affect the stored controller")
			}
			if scenario == "discard replacement" {
				if len(controller.Graveyard()) != 0 || len(controller.Library()) != 1 || controller.Library()[0].ID() != drawn.ID() {
					t.Error("discard replacement must put the drawn card in the library")
				}
			} else if len(controller.Library()) != 0 || len(controller.Graveyard()) != 1 || controller.Graveyard()[0].ID() != drawn.ID() {
				t.Error("the card drawn from an empty hand must then be discarded")
			}
		})
	}
}
