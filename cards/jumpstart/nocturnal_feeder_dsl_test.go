package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestNocturnalFeederDeathResolution(t *testing.T) {
	for _, scenario := range []string{"normal", "opponent controller", "source returns", "double gain", "prevent gain"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			controller, opponent := a, b
			if scenario == "opponent controller" {
				controller, opponent = b, a
			}
			watcher := g.PutOnBattlefield(dsl.NewArtifact("Watcher", "",
				dsl.WithAbility(dsl.WhenEvent(
					dsl.OpponentLosesLife(dsl.BindPlayer("opponent")),
					dsl.AddCounters(dsl.This, core.Charge, dsl.Fixed(1)),
				)),
			), controller.PlayerID())
			wantGain := 2
			if scenario == "double gain" {
				wantGain = 4
			}
			if scenario == "prevent gain" {
				wantGain = 0
			}
			g.AddLifeGainModifier(watcher.ID(), func(_ *mage.Game, player uuid.UUID, amount int) int {
				if player != controller.PlayerID() {
					t.Error("gain applied to wrong player")
				}
				if opponent.Life() != 18 {
					t.Errorf("opponent life before gain = %d, want 18", opponent.Life())
				}
				if amount != 2 {
					t.Errorf("requested gain = %d, want 2", amount)
				}
				return wantGain
			})
			card, err := mage.CreateCard("Nocturnal Feeder")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, controller.PlayerID())
			g.DestroyPermanent(g.FindPermanent(card.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("expected one death trigger")
			}
			if len(g.StackPeek().Targets) != 0 {
				t.Error("death trigger must have no targets")
			}
			if g.StackPeek().Controller != controller.PlayerID() {
				t.Fatal("death trigger has the wrong controller")
			}
			g.AssertLife(gametest.PlayerA, 20)
			g.AssertLife(gametest.PlayerB, 20)
			if scenario == "source returns" {
				returned, ok := a.RemoveFromGraveyard(card.ID())
				if !ok {
					t.Fatal("source is absent from its owner's graveyard")
				}
				g.PutOnBattlefield(returned, opponent.PlayerID())
			}
			g.ResolveStack()
			if controller.Life() != 20+wantGain || opponent.Life() != 18 {
				t.Errorf("controller/opponent life = %d/%d, want %d/18", controller.Life(), opponent.Life(), 20+wantGain)
			}
			if got := g.PlayerLifeGainedThisTurn(controller.PlayerID()); got != wantGain {
				t.Errorf("life gain event amount = %d, want %d", got, wantGain)
			}
			if got := g.PlayerLifeGainedThisTurn(opponent.PlayerID()); got != 0 {
				t.Errorf("opponent life gain event amount = %d, want 0", got)
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if got := g.FindPermanent(watcher.ID()).Counters[core.Charge]; got != 1 {
				t.Errorf("life loss triggers = %d, want 1", got)
			}
		})
	}
}
