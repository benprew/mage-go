package fallen_empires

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestGoblinWarDrums_GrantsMenace(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin War Drums")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	// PlayerB's creature should NOT get menace
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Hill Giant", core.Menace, true)
	g.AssertHasAbility(gametest.PlayerB, "Hill Giant", core.Menace, false)
}

func TestThrullRetainer_BoostsPT(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thrull Retainer")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thrull Retainer", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Hill Giant is 3/3, +1/+1 from Thrull Retainer = 4/4
	g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 4, 4)
	g.AssertAttachedTo(gametest.PlayerA, "Thrull Retainer", "Hill Giant")
}

func TestBreedingPit_CreatesThrullToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Breeding Pit")
	// Provide mana to pay the upkeep cost on turn 2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	// End of turn 1 (PlayerA's end step) should create a Thrull token
	g.StopAt(2, core.PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Thrull", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Thrull", 0, 1)
}

func TestElvenFortress_BoostsBlocker(t *testing.T) {
	// {1}{G}: Target blocking creature gets +0/+1 until end of turn.
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elven Fortress")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest", 4)
	// PlayerA attacks with Hill Giant, PlayerB blocks with Grizzly Bears
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
	// PlayerB activates Elven Fortress to boost their blocking Grizzly Bears
	g.ActivateAbility(1, core.CombatDamage, gametest.PlayerB, "Elven Fortress", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	// Grizzly Bears was 2/2, got +0/+1 = 2/3, took 3 damage from Hill Giant = exactly lethal
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestFungalBloom_AddsSporeCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fungal Bloom")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thallid")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fungal Bloom", "Thallid")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Thallid", core.Spore, 2) // 1 from upkeep + 1 from Fungal Bloom
}

func TestNightSoil_CreatesSaproling(t *testing.T) {
	// {1}, Exile two creature cards from a single graveyard: Create a 1/1 green Saproling creature token.
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Night Soil")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	// Put two creature cards in opponent's graveyard
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Night Soil")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Saproling", 1, 1)
	// The two creature cards should be exiled
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 0)
}
