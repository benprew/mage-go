package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestBloodhunterBatEntryResolution(t *testing.T) {
	for _, scenario := range []string{"opponent", "self", "source leaves", "double gain", "prevent gain"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			target, targetName := b, "PlayerB"
			if scenario == "self" {
				target, targetName = a, "PlayerA"
			}
			modifier := mage.NewArtifact("Modifier", "")
			g.PutOnBattlefield(modifier, a.PlayerID())
			wantGain := 2
			if scenario == "double gain" {
				wantGain = 4
			}
			if scenario == "prevent gain" {
				wantGain = 0
			}
			g.AddLifeGainModifier(modifier.ID(), func(_ *mage.Game, player uuid.UUID, amount int) int {
				if player != a.PlayerID() {
					t.Error("gain applied to wrong player")
				}
				if target.Life() != 18 {
					t.Errorf("target life before gain = %d, want 18", target.Life())
				}
				if scenario == "double gain" {
					return amount * 2
				}
				if scenario == "prevent gain" {
					return 0
				}
				return amount
			})
			card, err := mage.CreateCard("Bloodhunter Bat")
			if err != nil {
				t.Fatal(err)
			}
			if len(card.CastTargets()) != 0 {
				t.Fatal("entry target belongs to the trigger")
			}
			card.SetOwner(a.PlayerID())
			g.ChooseTarget(gametest.PlayerA, targetName)
			g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack size = %d, want 1", g.StackSize())
			}
			if targets := g.StackPeek().Targets; len(targets) != 1 || targets[0] != target.PlayerID() {
				t.Fatalf("trigger targets = %v, want %v", targets, target.PlayerID())
			}
			g.AssertLife(gametest.PlayerA, 20)
			g.AssertLife(gametest.PlayerB, 20)
			if scenario == "source leaves" {
				g.DestroyPermanent(g.FindPermanent(card.ID()))
			}
			g.ResolveStack()
			wantA, wantB := 20+wantGain, 18
			if scenario == "self" {
				wantA, wantB = 18+wantGain, 20
			}
			g.AssertLife(gametest.PlayerA, wantA)
			g.AssertLife(gametest.PlayerB, wantB)
			if got := g.PlayerLifeGainedThisTurn(a.PlayerID()); got != wantGain {
				t.Errorf("life gain event amount = %d, want %d", got, wantGain)
			}
			if got := g.PlayerLifeGainedThisTurn(b.PlayerID()); got != 0 {
				t.Errorf("opponent life gain event amount = %d, want 0", got)
			}
		})
	}
}
