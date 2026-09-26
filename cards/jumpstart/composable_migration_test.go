package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestMigratedSelfMillUsesLibraryTop(t *testing.T) {
	for _, name := range []string{"Crow of Dark Tidings", "Wailing Ghoul"} {
		t.Run(name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			player := g.GetPlayer(gametest.PlayerA)
			top := mage.NewLand("Top")
			middle := mage.NewLand("Middle")
			bottom := mage.NewLand("Bottom")
			for _, card := range []mage.Card{top, middle, bottom} {
				card.SetOwner(player.PlayerID())
			}
			player.SetLibrary([]mage.Card{top, middle, bottom})
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, name)
			g.ResolveStack()
			if len(player.Library()) != 1 || player.Library()[0].ID() != bottom.ID() {
				t.Fatal("mill must remove the two top cards")
			}
			g.AssertGraveyardCount(gametest.PlayerA, "Top", 1)
			g.AssertGraveyardCount(gametest.PlayerA, "Middle", 1)
		})
	}
}

func TestStabWoundKeepsUpkeepPlayerAndCausesLifeLoss(t *testing.T) {
	g := gametest.NewTestGame(t)
	a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
	first := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	second := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	aura := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stab Wound")
	g.Attach(aura, first)
	g.AddPreventionShield(b.PlayerID(), 10)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("expected one upkeep trigger")
	}
	g.Attach(aura, second)
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 18 {
		t.Fatalf("life totals = %d/%d; want 20/18", a.Life(), b.Life())
	}
}
