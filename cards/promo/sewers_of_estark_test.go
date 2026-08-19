package promo

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestSewersOfEstarkAttackingCreatureCantBeBlocked(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sewers of Estark")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.DeclareAttackers, gametest.PlayerA, "Sewers of Estark", "Hill Giant")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 17)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestSewersOfEstarkPreventsDamageFromBlockerAndBlockedCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sewers of Estark")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(2, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
	g.Block(2, gametest.PlayerA, "Grizzly Bears", "Grizzly Bears")
	g.CastSpell(2, core.DeclareBlockers, gametest.PlayerA, "Sewers of Estark", "Grizzly Bears")
	g.StopAt(2, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerA, 17)
}

func TestSewersOfEstarkPreventsDamageFromEachCreatureBlockerIsBlocking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Two-Headed Giant of Foriys")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Sewers of Estark")
	g.Attack(1, gametest.PlayerA, "Hill Giant", "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Two-Headed Giant of Foriys", "Hill Giant")
	g.Block(1, gametest.PlayerB, "Two-Headed Giant of Foriys", "Grizzly Bears")
	g.CastSpell(1, core.DeclareBlockers, gametest.PlayerB, "Sewers of Estark", "Two-Headed Giant of Foriys")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Two-Headed Giant of Foriys", 1)
}
