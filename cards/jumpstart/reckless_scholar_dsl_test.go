package jumpstart

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRecklessScholarDrawDiscardPipeline(t *testing.T) {
	for _, scenario := range []string{"self", "opponent", "source leaves", "discard replacement"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			owner := g.GetPlayer(gametest.PlayerA)
			controller := g.GetPlayer(gametest.PlayerB)
			target := owner
			if scenario == "self" {
				target = controller
			}
			drawn := mage.NewArtifact("Drawn card", "")
			drawn.SetOwner(target.PlayerID())
			target.AddToLibrary(drawn)
			var events []core.EventType
			for _, eventType := range []core.EventType{core.EvtCardDrawn, core.EvtDiscard} {
				watcher := mage.NewTriggered(eventType, false, mage.DrawFor(mage.You, mage.Fixed(1))).
					SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
						events = append(events, evt.Type)
						if evt.PlayerID != target.PlayerID() {
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
					mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), target.PlayerID())
			}
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Reckless Scholar")
			g.FindPermanent(source).Card.SetOwner(owner.PlayerID())
			if err := g.ActivateAbilityByIndex(controller.PlayerID(), source, 0, []uuid.UUID{target.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source).Tapped {
				t.Fatal("activation did not pay its tap cost")
			}
			if g.StackSize() != 1 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{target.PlayerID()}) || len(events) != 0 || len(target.Library()) != 1 {
				t.Fatal("activation must store its target and defer the draw")
			}
			if scenario == "source leaves" {
				g.ExilePermanent(g.FindPermanent(source))
			}
			g.ResolveStack()
			if !slices.Equal(events, []core.EventType{core.EvtCardDrawn, core.EvtDiscard}) {
				t.Errorf("events = %v, want draw then discard", events)
			}
			if len(controller.Hand()) != 0 || len(owner.Hand()) != 0 {
				t.Error("draw and discard must leave both hands empty")
			}
			if scenario == "discard replacement" {
				if len(target.Graveyard()) != 0 || len(target.Library()) != 1 || target.Library()[0].ID() != drawn.ID() {
					t.Error("discard replacement must put the drawn card in the library")
				}
			} else if len(target.Library()) != 0 || len(target.Graveyard()) != 1 || target.Graveyard()[0].ID() != drawn.ID() {
				t.Error("the card drawn from an empty hand must then be discarded")
			}
		})
	}
}
