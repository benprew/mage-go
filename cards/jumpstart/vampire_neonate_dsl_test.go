package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestVampireNeonateResolution(t *testing.T) {
	for _, scenario := range []string{"normal", "source leaves", "double gain", "prevent gain", "no mana", "tapped"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vampire Neonate")
			modifier := g.PutOnBattlefield(mage.NewArtifact("Modifier", ""), a.PlayerID())
			wantGain := 1
			if scenario == "double gain" {
				wantGain = 2
			}
			if scenario == "prevent gain" {
				wantGain = 0
			}
			g.AddLifeGainModifier(modifier.ID(), func(_ *mage.Game, player uuid.UUID, amount int) int {
				if player != a.PlayerID() {
					t.Error("gain applied to wrong player")
				}
				if b.Life() != 19 {
					t.Errorf("opponent life before gain = %d, want 19", b.Life())
				}
				if scenario == "double gain" {
					return amount * 2
				}
				if scenario == "prevent gain" {
					return 0
				}
				return amount
			})
			if scenario != "no mana" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			if scenario == "tapped" {
				g.TapPermanent(g.FindPermanent(source))
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, nil)
			if scenario == "no mana" || scenario == "tapped" {
				if err == nil {
					t.Fatal("activation succeeded without payable costs")
				}
				if g.StackSize() != 0 || g.FindPermanent(source).Tapped != (scenario == "tapped") {
					t.Fatal("failed payment changed source or stack")
				}
				wantMana := 0
				if scenario == "tapped" {
					wantMana = 2
				}
				if a.ManaPool().TotalMana() != wantMana {
					t.Fatal("failed payment spent mana")
				}
				g.AssertLife(gametest.PlayerA, 20)
				g.AssertLife(gametest.PlayerB, 20)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("activation did not pay its costs")
			}
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("expected one untargeted ability on the stack")
			}
			g.AssertLife(gametest.PlayerA, 20)
			g.AssertLife(gametest.PlayerB, 20)
			if scenario == "source leaves" {
				g.DestroyPermanent(g.FindPermanent(source))
			}
			g.ResolveStack()
			g.AssertLife(gametest.PlayerA, 20+wantGain)
			g.AssertLife(gametest.PlayerB, 19)
			if got := g.PlayerLifeGainedThisTurn(a.PlayerID()); got != wantGain {
				t.Errorf("life gain event amount = %d, want %d", got, wantGain)
			}
			if got := g.PlayerLifeGainedThisTurn(b.PlayerID()); got != 0 {
				t.Errorf("opponent life gain event amount = %d, want 0", got)
			}
		})
	}
}
