package jumpstart

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

// Black tail: Vampire Neonate / Wailing Ghoul / Wight / Witch
// Plus red and green-start chunk.

func TestVampireNeonate_DrainOnActivation(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Vampire Neonate")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.SetLife(gametest.PlayerA, 20)
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Vampire Neonate")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
}

func TestWailingGhoul_MillsTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Wailing Ghoul")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Wailing Ghoul")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Plains", 2)
}

func TestWightOfPrecinctSix_BoostByOpponentGraveyardCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Wight of Precinct Six")
	g.AddCard(ZoneGraveyard, gametest.PlayerB, "Grizzly Bears", 2)
	g.AddCard(ZoneGraveyard, gametest.PlayerB, "Lightning Bolt")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Wight of Precinct Six", 3, 3)
}

func TestBeetleback_CreatesTwoGoblins(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Beetleback Chief")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Beetleback Chief")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 2)
}

func TestFlametongueKavu_DealsFourOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Flametongue Kavu")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Flametongue Kavu", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestForgeDevil_DealsOneToTargetAndYou(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Forge Devil")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Forge Devil", "Mons's Goblin Raiders")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mons's Goblin Raiders", 1)
	g.AssertLife(gametest.PlayerA, 19)
}

func TestGoblinChieftain_BoostsAndHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Goblin Chieftain")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Mons's Goblin Raiders", 2, 2)
	g.AssertHasAbility(gametest.PlayerA, "Mons's Goblin Raiders", Haste, true)
}

func TestRagebloodShaman_BoostsAndTrampleMinotaurs(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Rageblood Shaman")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Borderland Minotaur")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Borderland Minotaur", 5, 4)
	g.AssertHasAbility(gametest.PlayerA, "Borderland Minotaur", Trample, true)
}

func TestBorderlandMarauder_BoostsOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Borderland Marauder")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(2, gametest.PlayerA, "Borderland Marauder")
	g.StopAt(2, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
}

func TestKilnFiend_BoostsOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Kiln Fiend")
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Kiln Fiend", 4, 2)
}

func TestYoungPyromancer_CreatesElementalOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Young Pyromancer")
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elemental", 1)
}

func TestThermoAlchemist_DealsOneToEachOpponent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Thermo-Alchemist")
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(2, PrecombatMain, gametest.PlayerA, "Thermo-Alchemist")
	g.StopAt(2, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
}

func TestGrimLavamancer_PingForTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grim Lavamancer")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Plains", 2)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(2, PrecombatMain, gametest.PlayerA, "Grim Lavamancer", "Grizzly Bears")
	g.StopAt(2, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestAmbassadorOak_CreatesElfWarrior(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Ambassador Oak")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Ambassador Oak")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elf Warrior", 1)
}

func TestCarvenCaryatid_DrawsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Carven Caryatid")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Carven Caryatid")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
}

func TestBrindleShoat_CreatesBoarOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Brindle Shoat")
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Brindle Shoat")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Boar", 1)
}
