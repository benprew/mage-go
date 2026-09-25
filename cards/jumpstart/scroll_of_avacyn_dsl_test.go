package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestScrollOfAvacynResolution(t *testing.T) {
	for _, scenario := range []string{"angel", "no angel", "opponent angel", "angel leaves", "angel enters", "two angels", "double gain", "prevent gain", "no mana", "noncreature angel"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			scroll := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scroll of Avacyn")
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
			var angel uuid.UUID
			if scenario != "no angel" && scenario != "angel enters" {
				player := gametest.PlayerA
				if scenario == "opponent angel" {
					player = gametest.PlayerB
				}
				angel = g.AddCard(core.ZoneBattlefield, player, "Serra Angel")
			}
			if scenario == "noncreature angel" {
				g.RemoveFromBattlefield(g.FindPermanent(angel))
				card := mage.NewArtifact("Angel permanent", "", mage.WithSubTypes("Angel"))
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if scenario == "two angels" {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
			}
			if scenario == "double gain" || scenario == "prevent gain" {
				g.AddLifeGainModifier(angel, func(_ *mage.Game, _ uuid.UUID, amount int) int {
					g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
					if scenario == "prevent gain" {
						return 0
					}
					return amount * 2
				})
			}
			if scenario != "no mana" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), scroll, 0, nil)
			if scenario == "no mana" {
				if err == nil {
					t.Fatal("activation succeeded without mana")
				}
				if g.FindPermanent(scroll) == nil || g.StackSize() != 0 {
					t.Fatal("failed payment changed source or stack")
				}
				g.AssertHandSize(gametest.PlayerA, 0)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g.AssertGraveyardCount(gametest.PlayerA, "Scroll of Avacyn", 1)
			g.AssertHandSize(gametest.PlayerA, 0)
			g.AssertLife(gametest.PlayerA, 20)
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("activation did not pay mana")
			}
			if scenario == "angel leaves" {
				g.RemoveFromBattlefield(g.FindPermanent(angel))
			}
			if scenario == "angel enters" {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
			}
			g.ResolveStack()
			want := 5
			switch scenario {
			case "no angel", "opponent angel", "angel leaves", "prevent gain":
				want = 0
			case "double gain":
				want = 10
			}
			g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
			g.AssertLife(gametest.PlayerA, 20+want)
			g.AssertLife(gametest.PlayerB, 20)
			if got := g.PlayerLifeGainedThisTurn(a.PlayerID()); got != want {
				t.Errorf("life gain event amount = %d, want %d", got, want)
			}
			if got := g.PlayerLifeGainedThisTurn(b.PlayerID()); got != 0 {
				t.Errorf("opponent life gain event amount = %d", got)
			}
		})
	}
}
