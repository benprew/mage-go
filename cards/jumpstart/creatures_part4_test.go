package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/antiquities"
	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestNyxathid_LargerHandShrinksMore(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 6)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Nyxathid", 1, 1)
}
func TestNyxathid_PinsOpponentAndShrinksByHandSize(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 3)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	pidB := g.GetPlayer(gametest.PlayerB).PlayerID()
	perm := g.FindPermanentByName("Nyxathid", g.GetPlayer(gametest.PlayerA).PlayerID())
	if perm == nil {
		t.Fatal("Nyxathid not on battlefield")
	}
	if perm.ChosenPlayer != pidB {
		t.Errorf("ChosenPlayer: got %v, want PlayerB %v", perm.ChosenPlayer, pidB)
	}
	g.AssertPowerToughness(gametest.PlayerA, "Nyxathid", 4, 4)
}
func TestOctoprophet(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Octoprophet")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain", "Forest"}, nil)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Octoprophet")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Plains", "Mountain", "Forest")
}
func TestOgreSlumlord_RatTokenAndDeathtouch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ogre Slumlord")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Rat", 1)
	g.AssertHasAbility(gametest.PlayerA, "Rat", core.Deathtouch, true)
}
func TestOneirophage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oneirophage")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.StopAt(5, core.PostcombatMain)
	g.Execute()
	// PlayerA draws on turn 3 and turn 5 = 2 cards.
	g.AssertCounterCount(gametest.PlayerA, "Oneirophage", core.P1P1, 2)
}
func TestOonasBlackguard_AdditionalCounterAndCombatDiscard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oona's Blackguard")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Corpse Hauler") // Human Rogue 2/1
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Corpse Hauler")
	// Corpse Hauler enters with an additional +1/+1 counter (3/2), attacks PlayerB
	// for 3 combat damage. The combat-damage trigger fires once for each
	// +1/+1-countered creature dealing damage to that player → PlayerB discards 1.
	g.Attack(3, gametest.PlayerA, "Corpse Hauler")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Corpse Hauler", core.P1P1, 1)
	g.AssertHandCount(gametest.PlayerB, "Mountain", 1)
}
func TestOrneryGoblin_DealsOneToBlocker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornery Goblin")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Ornery Goblin", "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestOvergrownBattlement_AddsGreenForEachDefender(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Overgrown Battlement")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Vines")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Overgrown Battlement", core.Defender, true)
}
func TestPatronOfTheValiant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Patron of the Valiant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Patron of the Valiant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
}
func TestPenumbraBobcat_CreatesTokenOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Penumbra Bobcat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.Attack(1, gametest.PlayerA, "Penumbra Bobcat")
	g.Block(1, gametest.PlayerB, "Craw Wurm", "Penumbra Bobcat")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Cat", 1)
}
func TestPerilousMyr_DealsTwoOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Perilous Myr")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(1, gametest.PlayerA, "Perilous Myr")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Perilous Myr")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestPerilousMyr_DealsTwoOnDeath_PlayerTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Perilous Myr")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(1, gametest.PlayerA, "Perilous Myr")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Perilous Myr")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
}
func TestPhyrexianBroodlings_SacForCounter(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Broodlings")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Broodlings")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Phyrexian Broodlings", 3, 3)
}
func TestPhyrexianDebaser_SacToDebuff(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Debaser")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Debaser", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 1, 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Phyrexian Debaser", 1)
}
func TestPhyrexianGargantua_DrawTwoLoseTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Phyrexian Gargantua")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Gargantua")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 2)
	g.AssertLife(gametest.PlayerA, 18)
}
func TestPhyrexianRager_DrawAndLose(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Phyrexian Rager")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Rager")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertLife(gametest.PlayerA, 19)
}
func TestPlaguedRusalka_SacToDebuff(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plagued Rusalka")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Plagued Rusalka", "Hill Giant")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 2, 2)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestPouncingCheetah_Flash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Pouncing Cheetah")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest", 3)
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Pouncing Cheetah")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Pouncing Cheetah", 1)
}
func TestPrescientChimera(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Prescient Chimera")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
	g.ChooseScry(gametest.PlayerA, []string{"Forest"}, nil)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertLibraryTop(gametest.PlayerA, "Island", "Forest")
}
func TestPrimordialSage_DrawsOnCreatureCast(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primordial Sage")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Leaf Gilder")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Leaf Gilder")
	g.StopAt(1, core.EndStep)
	g.Execute()
}
func TestProsperousPirates_CreatesTwoTreasuresOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Prosperous Pirates")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Prosperous Pirates")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Prosperous Pirates", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 2)
}
func TestPyroclasticElemental_PingsPlayer(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pyroclastic Elemental")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pyroclastic Elemental", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
}
func TestRagebloodShaman_BoostsAndTrampleMinotaurs(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rageblood Shaman")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Borderland Minotaur")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Borderland Minotaur", 5, 4)
	g.AssertHasAbility(gametest.PlayerA, "Borderland Minotaur", core.Trample, true)
}
func TestRagingRegisaur_PingOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging Regisaur")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(1, gametest.PlayerA, "Raging Regisaur")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	// 4 (combat) + 1 (trigger) = 5
	g.AssertLife(gametest.PlayerB, 15)
}
func TestRagingRegisaur_PingTargetsCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging Regisaur")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Raging Regisaur")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
}
func TestRapaciousDragon_CreatesTwoTreasuresOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rapacious Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rapacious Dragon")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Rapacious Dragon", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 2)
	g.AssertHasAbility(gametest.PlayerA, "Rapacious Dragon", core.Flying, true)
}
func TestRattlechains(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rattlechains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Will-o'-the-Wisp") // Spirit
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rattlechains", "Will-o'-the-Wisp")
	g.StopAt(1, core.DeclareAttackers)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Will-o'-the-Wisp", core.Hexproof, true)
}
func TestRattlechainsFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Rattlechains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Will-o'-the-Wisp")
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Rattlechains", "Will-o'-the-Wisp")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Rattlechains", 1)
}
func TestRavenousBaloth_GainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ravenous Baloth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rumbling Baloth")
	g.SetLife(gametest.PlayerA, 20)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ravenous Baloth")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 24)
}
func TestRavenousChupacabra_DestroysOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ravenous Chupacabra")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ravenous Chupacabra", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestRecklessScholar(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Reckless Scholar")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 10)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Reckless Scholar", "PlayerA")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
}

// Rhox Faithmender: doubles life gain.
func TestRhoxFaithmender_DoublesLifegain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rhox Faithmender")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 1)
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.ChooseMode(gametest.PlayerA, 0)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 26)
}
func TestRishkarPeemaRenegade_ETBCountersAndManaGrant(t *testing.T) {
	t.Run("ETB places counter on chosen target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("ETB with zero targets resolves cleanly", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Rishkar, Peema Renegade", 1)
	})
}
func TestRonomUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ronom Unicorn")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Holy Strength") // an enchantment
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // for Holy Strength to attach
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Ronom Unicorn", "Holy Strength")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Holy Strength", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Ronom Unicorn", 1)
}
func TestRunedServitor_EachPlayerDraws(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Runed Servitor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest", 5)
	g.Attack(1, gametest.PlayerA, "Runed Servitor")
	g.Block(1, gametest.PlayerB, "Craw Wurm", "Runed Servitor")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Runed Servitor", 1)
}
func TestSagesRowSavant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sage's Row Savant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, nil, []string{"Forest", "Mountain"})
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sage's Row Savant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Mountain", "Plains")
}
func TestSailorOfMeans_CreatesTreasureOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sailor of Means")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sailor of Means")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Sailor of Means", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 1)
}
func TestSailorOfMeans_TreasureSacAddsAnyMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sailor of Means")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sailor of Means")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Treasure")
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 0)
	g.AssertManaProducedAtLeast(gametest.PlayerA, core.Blue, 1)
}
func TestSangromancer_GainsOnOpponentCreatureDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sangromancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
}
func TestSangromancer_GainsOnOpponentDiscard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sangromancer")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Twist")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mind Twist", 1, "PlayerB")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
}

// Scrounging Bandar: at upkeep, may move +1/+1 counters to another creature.
func TestScroungingBandar_MovesCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scrounging Bandar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Scrounging Bandar", core.P1P1, 2)
	g.ChooseNumber(gametest.PlayerA, 2)
	g.StopAt(2, core.PrecombatMain)
	g.Execute()
	// Bandar moved both counters away → became 0/0 → died as SBA.
	g.AssertPermanentCount(gametest.PlayerA, "Scrounging Bandar", 0)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
}
func TestSeaGateOracle(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sea Gate Oracle")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sea Gate Oracle")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertLibraryCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestSeismicElemental_PreventsNonflyersBlocking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Seismic Elemental")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Seismic Elemental")
	g.Attack(3, gametest.PlayerA, "Seismic Elemental")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
}

// Selvala, Heart of the Wilds: {G}, {T}: add X mana of any combination, where X
// is greatest power among creatures you control.
func TestSelvala_AddsXManaInAnyCombination(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	// Hill Giant is 3/3 — greatest power among creatures Selvala's controller has = 3.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.ChooseManaColor(gametest.PlayerA, core.Red)
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.ChooseManaColor(gametest.PlayerA, core.Green)
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertManaProducedAtLeast(gametest.PlayerA, core.Red, 1)
	g.AssertManaProducedAtLeast(gametest.PlayerA, core.Blue, 1)
	g.AssertManaProducedAtLeast(gametest.PlayerA, core.Green, 1)
}

// Selvala, Heart of the Wilds: when another creature with greater power
// enters, draw a card.
func TestSelvalaHeartOfTheWilds_ETBDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "War Mammoth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 3)
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "War Mammoth")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mountain", 1)
}

// Selvala — controller may decline the optional draw.
func TestSelvalaHeartOfTheWilds_MayDecline(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}

// Selvala — entering creature opponent controls has greatest power: opponent draws.
func TestSelvalaHeartOfTheWilds_OpponentDrawsOnTheirBigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 3)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// PlayerB is the controller of the entering Hill Giant, so PlayerB draws
	// from the Selvala trigger — one Plains in hand.
	g.AssertHandCount(gametest.PlayerB, "Plains", 1)
	// PlayerA (Selvala's controller) must NOT have been the one to draw.
	g.AssertHandCount(gametest.PlayerA, "Forest", 0)
}

// Selvala — tied power (Bears 2 vs Selvala 2) is not strictly greater: no draw.
func TestSelvalaHeartOfTheWilds_TiedPowerNoDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}

// Selvala — entering creature ties an existing larger creature: no draw.
func TestSelvalaHeartOfTheWilds_TiesExistingLargest(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "War Mammoth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "War Mammoth")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// War Mammoth is 3/3, Hill Giant is 3/3 — not strictly greater.
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}

// Selvala — entering creature you control has greatest power: you draw.
func TestSelvalaHeartOfTheWilds_YouDrawOnYourBigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	// The Selvala trigger fires when Hill Giant enters (Hill Giant power 3 > Selvala 2 and Bears 2),
	// drawing one extra card for PlayerA.
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
}
func TestSethronHurloonGeneral_PumpMinotaursWithB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sethron, Hurloon General")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sethron, Hurloon General")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Sethron, Hurloon General", 5, 4)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", core.Menace, true)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", core.Haste, true)
}
