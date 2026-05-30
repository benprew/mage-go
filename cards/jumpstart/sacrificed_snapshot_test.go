package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	. "github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// TestFling_DealsDamageEqualToSacrificedPower verifies Fling deals
// damage equal to the sacrificed creature's power to any target.
func TestFling_DealsDamageEqualToSacrificedPower(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant") // power 3
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(ZoneHand, gametest.PlayerA, "Fling")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Fling", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Fling", 1)
}

// TestMomentousFall_DrawsAndGainsLife verifies Momentous Fall draws
// cards equal to the sacrificed creature's power and gains life equal
// to its toughness.
func TestMomentousFall_DrawsAndGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	// Hill Giant is 3/3
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Momentous Fall")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Momentous Fall")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Momentous Fall", 1)
	// Hill Giant is 3/3 → +3 life
	g.AssertLife(gametest.PlayerA, 23)
}

// TestLena_SacGivesIndestructibleToSmallerCreatures verifies that
// sacrificing Lena grants indestructible until end of turn to creatures
// you control with power less than Lena's power.
func TestLena_SacGivesIndestructibleToSmallerCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Lena, Selfless Champion") // 3/3
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")           // 2/2 — smaller
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant")              // 3/3 — equal, not smaller
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Lena, Selfless Champion")
	g.StopAt(1, EndStep)
	g.Execute()

	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", Indestructible, true)
	g.AssertHasAbility(gametest.PlayerA, "Hill Giant", Indestructible, false)
	g.AssertGraveyardCount(gametest.PlayerA, "Lena, Selfless Champion", 1)
}

// TestGhoulcallerGisa_CreatesZombiesEqualToSacPower verifies Gisa
// creates X 2/2 black Zombie tokens where X is the sacrificed
// creature's power.
func TestGhoulcallerGisa_CreatesZombiesEqualToSacPower(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Ghoulcaller Gisa")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Ghoulcaller Gisa")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Zombie", 3)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
}
