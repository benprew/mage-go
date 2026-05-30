package jumpstart

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Tests for cards wired to the multi-target ReturnFromGraveyardToHandTarget
// effect.

func TestSoulSalvage_ReturnsTwoCreatureCards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Soul Salvage")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Hill Giant")

	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Soul Salvage", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, EndStep)
	g.Execute()

	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
}

func TestMacabreWaltz_ReturnsTwoThenDiscards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Macabre Waltz")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Hill Giant")

	g.ChooseDiscard(gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Macabre Waltz", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, EndStep)
	g.Execute()

	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Macabre Waltz", 1)
}
