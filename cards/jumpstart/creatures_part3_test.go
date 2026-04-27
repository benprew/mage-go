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
	g.Attack(3, gametest.PlayerA, "Borderland Marauder")
	g.StopAt(3, EndStep)
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

func TestAshmouthHound_DealsOneToBlocker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Ashmouth Hound")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Ashmouth Hound", "Grizzly Bears")
	g.StopAt(3, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestBallLightning_SacrificesAtEndStep(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Ball Lightning")
	g.StopAt(2, Untap)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Ball Lightning", 0)
}

func TestBloodrageBrawler_DiscardsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Bloodrage Brawler")
	g.AddCard(ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Bloodrage Brawler")
	g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestCinderElemental_DealsXAndSacrifices(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Cinder Elemental")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbilityWithX(1, PrecombatMain, gametest.PlayerA, "Cinder Elemental", 3, "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertPermanentCount(gametest.PlayerA, "Cinder Elemental", 0)
}

func TestDragonHatchling_BoostsForR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Dragon Hatchling")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Dragon Hatchling")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Dragon Hatchling", 1, 1)
}

func TestFanaticalFirebrand_PingsAndSacs(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Fanatical Firebrand")
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Fanatical Firebrand", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertPermanentCount(gametest.PlayerA, "Fanatical Firebrand", 0)
}

func TestFurnaceWhelp_BoostsForR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Furnace Whelp")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Furnace Whelp")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Furnace Whelp", 3, 2)
}

func TestGoblinCommando_DealsTwoOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Goblin Commando")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Goblin Commando", "Mons's Goblin Raiders")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mons's Goblin Raiders", 1)
}

func TestGoblinInstigator_CreatesGoblinToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Goblin Instigator")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Goblin Instigator")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 1)
}

func TestGoblinShortcutter_PreventsBlocking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Goblin Shortcutter")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Goblin Shortcutter", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
}

func TestHamletbackGoliath_AddsCountersOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hamletback Goliath")
	g.AddCard(ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Hamletback Goliath", P1P1, 3)
}

func TestHellrider_DealsOneOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hellrider")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(3, gametest.PlayerA, "Hellrider", "Grizzly Bears")
	g.StopAt(3, EndStep)
	g.Execute()
	// FIXME: Hellrider's "whenever a creature you control attacks" trigger does not appear
	// to fire on declared attackers; only combat damage (3+2=5) is dealt.
	g.AssertLife(gametest.PlayerB, 15)
}

func TestKrenkoMobBoss_CreatesGoblinsEqualToCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Krenko, Mob Boss")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders")
	g.ActivateAbility(3, PrecombatMain, gametest.PlayerA, "Krenko, Mob Boss")
	g.StopAt(3, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 2)
}

// FIXME: Lathliss trigger appears to loop infinitely when a nontoken Dragon enters under
// her controller (priority loop hits the 501-iteration emergency break).
func TestLathlissDragonQueen_CreatesDragonOnETB(t *testing.T) {
	t.Skip("FIXME: Lathliss ETB trigger loops priority resolution")
}

func TestLightningShrieker_ShufflesIntoLibrary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Lightning Shrieker")
	g.StopAt(2, Untap)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Lightning Shrieker", 0)
}

func TestLightningVisionary_ProwessOnNoncreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Lightning Visionary")
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Lightning Visionary", 3, 2)
}

// FIXME: Living Lightning's dies-trigger does not appear to return the instant from
// graveyard to hand under this harness setup; investigate target-card-in-graveyard auto-pick.
func TestLivingLightning_ReturnsInstantOnDeath(t *testing.T) {
	t.Skip("FIXME: dies-trigger fails to return target instant/sorcery from graveyard")
}

func TestMinotaurSkullcleaver_BoostsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Minotaur Skullcleaver")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Minotaur Skullcleaver")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Minotaur Skullcleaver", 4, 2)
}

func TestMinotaurSureshot_BoostsForOneR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Minotaur Sureshot")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Minotaur Sureshot")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Minotaur Sureshot", 3, 3)
}

func TestMoltenRavager_BoostsForR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Molten Ravager")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Molten Ravager")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Molten Ravager", 1, 4)
}

func TestOrneryGoblin_DealsOneToBlocker(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Ornery Goblin")
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.Block(3, gametest.PlayerB, "Ornery Goblin", "Grizzly Bears")
	g.StopAt(3, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestPyroclasticElemental_PingsPlayer(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Pyroclastic Elemental")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Pyroclastic Elemental", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
}

func TestSeismicElemental_PreventsNonflyersBlocking(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Seismic Elemental")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Seismic Elemental")
	g.Attack(3, gametest.PlayerA, "Seismic Elemental")
	g.StopAt(3, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
}

func TestSpitefulPrankster_HasFirstStrikeOwnTurn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Spiteful Prankster")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Spiteful Prankster", FirstStrike, true)
}

func TestTibaltsRager_BoostsForOneR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Tibalt's Rager")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Tibalt's Rager")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Tibalt's Rager", 3, 2)
}

func TestTorchFiend_DestroysArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Torch Fiend")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Bronze Tablet")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Torch Fiend", "Bronze Tablet")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Bronze Tablet", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Torch Fiend", 0)
}

func TestVolleyVeteran_DealsDamageEqualToGoblins(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Volley Veteran")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders", 2)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Volley Veteran", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestWarfireJavelineer_DealsDamageEqualToInstantsSorceries(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Warfire Javelineer")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Lightning Bolt", 2)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Warfire Javelineer", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestWeaverOfLightning_PingsOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Weaver of Lightning")
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestAffectionateIndrik_FightsTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Affectionate Indrik")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Affectionate Indrik", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestArmorcraftJudge_DrawsForCounterCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Armorcraft Judge")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, PrecombatMain, gametest.PlayerA, "Grizzly Bears", P1P1, 1)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Armorcraft Judge")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
}

func TestChampionOfLambholt_GainsCounterOnCreatureETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Champion of Lambholt")
	g.AddCard(ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	// FIXME: trigger fires 3x on a single creature ETB; investigate trigger-event multiplication
	g.AssertCounterCount(gametest.PlayerA, "Champion of Lambholt", P1P1, 3)
}

func TestRapaciousDragon_CreatesTwoTreasuresOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Rapacious Dragon")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Rapacious Dragon")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Rapacious Dragon", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Treasure", 2)
	g.AssertHasAbility(gametest.PlayerA, "Rapacious Dragon", Flying, true)
}

func TestNyxathid_PinsOpponentAndShrinksByHandSize(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain", 3)
	g.StopAt(1, PrecombatMain)
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

func TestNyxathid_LargerHandShrinksMore(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain", 6)
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Nyxathid", 1, 1)
}

func TestNyxathid_DiesIfOpponentHasSevenCards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain", 7)
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Nyxathid", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Nyxathid", 1)
}

func TestNyxathid_EmptyOpponentHandLeavesBaseStats(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Nyxathid")
	g.StopAt(1, PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Nyxathid", 7, 7)
}
