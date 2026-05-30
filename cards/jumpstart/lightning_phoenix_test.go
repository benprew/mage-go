package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Lightning Phoenix {2}{R}
// Creature — Phoenix  2/2
// Flying, haste
// This creature can't block.
// At the beginning of your end step, if an opponent was dealt 3 or more damage
// this turn, you may pay {R}. If you do, return this card from your graveyard
// to the battlefield.

func TestLightningPhoenix_StatsAndKeywords(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightning Phoenix")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Lightning Phoenix", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Lightning Phoenix", 2, 2)
	g.AssertHasAbility(gametest.PlayerA, "Lightning Phoenix", core.Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Lightning Phoenix", core.Haste, true)
}

func TestLightningPhoenix_CantBlock(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightning Phoenix")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")
	g.Block(2, gametest.PlayerA, "Lightning Phoenix", "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// Block was rejected; Bears' 2 damage gets through.
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertPermanentCount(gametest.PlayerA, "Lightning Phoenix", 1)
}

func TestLightningPhoenix_GraveyardReturnAfterOpponent3Damage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Phoenix")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.Cleanup)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Lightning Phoenix", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Phoenix", 0)
	g.AssertLife(gametest.PlayerB, 17)
}

func TestLightningPhoenix_NoReturnWhenDamageBelowThreshold(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Phoenix")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
	g.StopAt(1, core.Cleanup)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Phoenix", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Lightning Phoenix", 0)
	g.AssertLife(gametest.PlayerB, 18)
}

func TestLightningPhoenix_DeclineMayPay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Phoenix")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.StopAt(1, core.Cleanup)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Phoenix", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Lightning Phoenix", 0)
}
