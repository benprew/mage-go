package arabian

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestFishliverOil(t *testing.T) {
	t.Run("grants_islandwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fishliver Oil")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 Defender
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fishliver Oil", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Islandwalk — can't be blocked (defender has Island)
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("blockable_without_island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fishliver Oil")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")
		// No Island for defender
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fishliver Oil", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Can be blocked normally
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestUnstableMutation(t *testing.T) {
	t.Run("immediate_boost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men") // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unstable Mutation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unstable Mutation", "Flying Men")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Flying Men", 4, 4) // 1+3, 1+3
	})

	t.Run("after_one_upkeep_gets_counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men") // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unstable Mutation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unstable Mutation", "Flying Men")
		g.StopAt(3, core.PrecombatMain) // next PlayerA upkeep: turn 3
		g.Execute()
		// +3/+3 from mutation, -1/-1 from one counter = net +2/+2
		g.AssertPowerToughness(gametest.PlayerA, "Flying Men", 3, 3)
	})
}
