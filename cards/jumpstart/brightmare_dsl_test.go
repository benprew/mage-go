package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestBrightmareEntryTrigger(t *testing.T) {
	for _, scenario := range []string{"self", "opponent", "already tapped", "zero targets", "source leaves", "target returns", "power changes", "negative power"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Brightmare")
			if err != nil {
				t.Fatal(err)
			}
			if len(card.CastTargets()) != 0 {
				t.Fatal("entry targets must belong to the trigger")
			}
			card.SetOwner(a.PlayerID())
			targetID := card.ID()
			targetName := card.Name()
			wantLife := 22
			if scenario != "self" {
				targetID = g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
				targetName = "Hill Giant"
				wantLife = 23
			}
			if scenario == "already tapped" {
				g.TapPermanent(g.FindPermanent(targetID))
			}
			if scenario != "zero targets" {
				g.ChooseTarget(gametest.PlayerA, targetName)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack size = %d, want one entry trigger", g.StackSize())
			}
			g.AssertLife(gametest.PlayerA, 20)
			switch scenario {
			case "zero targets":
				if len(g.StackPeek().Targets) != 0 {
					t.Fatal("expected no targets")
				}
				wantLife = 20
			case "source leaves":
				g.DestroyPermanent(g.FindPermanent(card.ID()))
			case "target returns":
				targetCard := g.FindPermanent(targetID).Card
				g.RemoveFromBattlefield(g.FindPermanent(targetID))
				g.PutOnBattlefield(targetCard, b.PlayerID())
				wantLife = 20
			case "power changes", "negative power":
				delta := 2
				wantLife = 25
				if scenario == "negative power" {
					delta = -5
					wantLife = 20
				}
				g.AddContinuousEffect(mage.TemporaryBoost(targetID, delta, 0))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			g.AssertLife(gametest.PlayerA, wantLife)
			g.AssertLife(gametest.PlayerB, 20)
			wantTapped := scenario != "zero targets" && scenario != "target returns"
			if g.FindPermanent(targetID).Tapped != wantTapped {
				t.Fatalf("target tapped = %v, want %v", g.FindPermanent(targetID).Tapped, wantTapped)
			}
		})
	}
}
