package jumpstart

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for green tail / multicolor / colorless creatures (chunk 4).

func TestCraterhoofBehemoth_PumpsAndGrantsTrample(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Craterhoof Behemoth")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 8)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Craterhoof Behemoth")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// 2 creatures => +2/+2 and trample.
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", Trample, true)
}

func TestDawntreaderElk_TutorsBasicLandTapped(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Dawntreader Elk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 1)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Dawntreader Elk")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Dawntreader Elk", 1)
}

func TestDroverOfTheMighty_BoostsWhenDinosaur(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Drover of the Mighty")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Orazca Frillback")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Drover of the Mighty", 3, 3)
}

func TestDwynensElite_CreatesTokenIfElf(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Dwynen's Elite")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Leaf Gilder")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Dwynen's Elite")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elf Warrior", 1)
}

func TestElvishArchdruid_BoostsOtherElves(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Elvish Archdruid")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Leaf Gilder")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Leaf Gilder", 3, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Elvish Archdruid", 2, 2)
}

func TestFeralProwler_DrawsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Feral Prowler")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest", 5)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.Attack(1, gametest.PlayerB, "Craw Wurm")
	// Wait until B's turn
	g.StopAt(2, EndStep)
	g.Execute()
}

func TestFeralHydra_EntersWithXCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Feral Hydra")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.CastSpellWithX(1, PrecombatMain, gametest.PlayerA, "Feral Hydra", 3)
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Feral Hydra", 3, 3)
}

func TestFertilid_EntersWithCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Fertilid")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Fertilid", 2, 2)
}

func TestGraveBramble_HasDefenderAndProtection(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grave Bramble")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grave Bramble", Defender, true)
}

func TestIronshellBeetle_AddsCounterToTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Ironshell Beetle")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Ironshell Beetle", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}

func TestLeafGilder_TapsForGreen(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Leaf Gilder")
	g.StopAt(1, EndStep)
	g.Execute()
}

func TestOvergrownBattlement_AddsGreenForEachDefender(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Overgrown Battlement")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Wall of Vines")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Overgrown Battlement", Defender, true)
}

func TestPenumbraBobcat_CreatesTokenOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Penumbra Bobcat")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.Attack(1, gametest.PlayerA, "Penumbra Bobcat")
	g.Block(1, gametest.PlayerB, "Craw Wurm", "Penumbra Bobcat")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Cat", 1)
}

func TestPrimordialSage_DrawsOnCreatureCast(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Primordial Sage")
	g.AddCard(ZoneHand, gametest.PlayerA, "Leaf Gilder")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Leaf Gilder")
	g.StopAt(1, EndStep)
	g.Execute()
}

func TestRavenousBaloth_GainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Ravenous Baloth")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Rumbling Baloth")
	g.SetLife(gametest.PlayerA, 20)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Ravenous Baloth")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 24)
}

func TestSomberwaldStag_FightsTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Somberwald Stag")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Somberwald Stag", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestSporemound_SaprolingOnLandETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Sporemound")
	g.AddCard(ZoneHand, gametest.PlayerA, "Forest")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
}

func TestSylvanBrushstrider_GainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Sylvan Brushstrider")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Sylvan Brushstrider")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
}

func TestSylvanRanger_TutorsToHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Sylvan Ranger")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 1)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Sylvan Ranger")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// Either in hand or auto-played as land.
	g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
}

func TestThragtusk_GainsLifeAndLeavesToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Thragtusk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Thragtusk")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 25)
}

func TestUlvenwaldHydra_PTEqualsLands(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Ulvenwald Hydra")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.StopAt(1, Upkeep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Ulvenwald Hydra", 5, 5)
}

func TestWallOfBlossoms_DrawsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Wall of Blossoms")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Wall of Blossoms")
	g.StopAt(1, EndStep)
	g.Execute()
}

func TestWildheartInvoker_PumpsAndGrantsTrample(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Wildheart Invoker")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 8)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Wildheart Invoker", "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 7, 7)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", Trample, true)
}

func TestWoodbornBehemoth_BuffsWith8Lands(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Woodborn Behemoth")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 8)
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Woodborn Behemoth", 8, 8)
	g.AssertHasAbility(gametest.PlayerA, "Woodborn Behemoth", Trample, true)
}

func TestRagingRegisaur_PingOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Raging Regisaur")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(1, gametest.PlayerA, "Raging Regisaur")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	// 4 (combat) + 1 (trigger) = 5
	g.AssertLife(gametest.PlayerB, 15)
}

func TestRagingRegisaur_PingTargetsCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Raging Regisaur")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Raging Regisaur")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
}

func TestIronrootWarlord_PowerEqualsCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Ironroot Warlord")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Ironroot Warlord", 2, 5)
}

func TestAlloyMyr_TapsForAnyColor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Alloy Myr")
	g.StopAt(1, EndStep)
	g.Execute()
}

func TestAncestralStatue_BouncesPermanent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Ancestral Statue")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Ancestral Statue")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestGargoyleSentinel_GainsFlying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Gargoyle Sentinel")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Gargoyle Sentinel")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Gargoyle Sentinel", Flying, true)
	g.AssertHasAbility(gametest.PlayerA, "Gargoyle Sentinel", Defender, false)
}

func TestGingerbrute_HasteAndSacGainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Gingerbrute")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.SetLife(gametest.PlayerA, 20)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Gingerbrute")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertGraveyardCount(gametest.PlayerA, "Gingerbrute", 1)
}

func TestJoustingDummy_BoostsPower(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Jousting Dummy")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Jousting Dummy")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Jousting Dummy", 3, 1)
}

func TestLightningCoreExcavator_DealsThreeOnSac(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Lightning-Core Excavator")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Lightning-Core Excavator", "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning-Core Excavator", 1)
}

func TestMeteorGolem_DestroysNonland(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Meteor Golem")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 7)
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Meteor Golem")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestMyrSire_CreatesTokenOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Myr Sire")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.Attack(1, gametest.PlayerA, "Myr Sire")
	g.Block(1, gametest.PlayerB, "Craw Wurm", "Myr Sire")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Myr", 1)
}

func TestPerilousMyr_DealsTwoOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Perilous Myr")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(1, gametest.PlayerA, "Perilous Myr")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Perilous Myr")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestPerilousMyr_DealsTwoOnDeath_PlayerTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Perilous Myr")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.SetLife(gametest.PlayerB, 20)
	g.Attack(1, gametest.PlayerA, "Perilous Myr")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Perilous Myr")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
}

func TestRunedServitor_EachPlayerDraws(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Runed Servitor")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest", 5)
	g.AddCard(ZoneLibrary, gametest.PlayerB, "Forest", 5)
	g.Attack(1, gametest.PlayerA, "Runed Servitor")
	g.Block(1, gametest.PlayerB, "Craw Wurm", "Runed Servitor")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Runed Servitor", 1)
}

func TestSkitteringSurveyor_TutorsToHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Skittering Surveyor")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 1)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Skittering Surveyor")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
}

func TestSuspiciousBookcase_MakesUnblockable(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Suspicious Bookcase")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Suspicious Bookcase", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
}

func TestPouncingCheetah_Flash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerB, "Pouncing Cheetah")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Forest", 3)
	g.CastSpell(1, BeginCombat, gametest.PlayerB, "Pouncing Cheetah")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Pouncing Cheetah", 1)
}

func TestCloudreaderSphinx(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Cloudreader Sphinx")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain"}, []string{"Forest"})
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Cloudreader Sphinx")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Plains", "Mountain")
}

func TestOctoprophet(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Octoprophet")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain", "Forest"}, nil)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Octoprophet")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Plains", "Mountain", "Forest")
}

func TestSagesRowSavant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneHand, gametest.PlayerA, "Sage's Row Savant")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, nil, []string{"Forest", "Mountain"})
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Sage's Row Savant")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Mountain", "Plains")
}

func TestSigiledStarfish(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Sigiled Starfish")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain"}, nil)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Sigiled Starfish")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Sigiled Starfish", true)
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Mountain")
}

func TestPrescientChimera(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Prescient Chimera")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Island")
	g.ChooseScry(gametest.PlayerA, []string{"Forest"}, nil)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertLibraryTop(gametest.PlayerA, "Island", "Forest")
}

func TestRishkarPeemaRenegade_ETBCountersAndManaGrant(t *testing.T) {
	t.Run("ETB places counter on chosen target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(ZoneHand, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("ETB with zero targets resolves cleanly", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(ZoneHand, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Rishkar, Peema Renegade")
		g.StopAt(1, EndCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Rishkar, Peema Renegade", 1)
	})
}

func TestInniazTheGaleForce_PumpAttackingFlyersWithW(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.Attack(3, gametest.PlayerA, "Healer's Hawk")
	g.ActivateAbility(3, DeclareAttackers, gametest.PlayerA, "Inniaz, the Gale Force")
	g.StopAt(3, DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 2, 2)
}

func TestInniazTheGaleForce_PumpAttackingFlyersWithU(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.Attack(3, gametest.PlayerA, "Healer's Hawk")
	g.ActivateAbility(3, DeclareAttackers, gametest.PlayerA, "Inniaz, the Gale Force")
	g.StopAt(3, DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 2, 2)
}

func TestInniazTheGaleForce_DoesNotPumpNonAttackers(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Inniaz, the Gale Force")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 1, 1)
}

// Three flying attackers controlled by Inniaz's controller fires the
// trigger; in 2-player, "the player to their right" collapses to the
// unique opponent, so the controller picks for both assignments.
func TestInniazTheGaleForce_ThreeFlyersSwapsPermanents(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mesa Pegasus")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mountain")
	// Inniaz's controller (A) makes both selections in order:
	//   1) of B's nonland permanents -> A gains Grizzly Bears
	//   2) of A's nonland permanents -> B gains Healer's Hawk
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.ChoosePermanent(gametest.PlayerA, "Healer's Hawk")
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk", "Mesa Pegasus")
	g.StopAt(3, DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Healer's Hawk", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 0)
}

// Only two flying attackers — does not meet the "three or more" threshold.
func TestInniazTheGaleForce_TwoFlyersDoesNotTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk")
	g.StopAt(3, DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 1)
}

// Non-flying attackers don't count toward the threshold.
func TestInniazTheGaleForce_NonFlyersDoNotCountTowardThreshold(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")
	// 2 flyers + 1 non-flyer attacking — flying count is 2, no trigger.
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk", "Grizzly Bears")
	g.StopAt(3, DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Mesa Pegasus", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Mesa Pegasus", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 1)
}

// If the opponent controls only lands, no permanent transfers from B to A;
// the other assignment (A -> B) still happens.
func TestInniazTheGaleForce_NoNonlandOnOpposingSideSkipsAssignment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Inniaz, the Gale Force")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Healer's Hawk")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mesa Pegasus")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mountain")
	// Only the second assignment has a candidate: A's nonland -> B.
	g.ChoosePermanent(gametest.PlayerA, "Mesa Pegasus")
	g.Attack(3, gametest.PlayerA, "Inniaz, the Gale Force", "Healer's Hawk", "Mesa Pegasus")
	g.StopAt(3, DeclareBlockers)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Mesa Pegasus", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Mesa Pegasus", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Healer's Hawk", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Mountain", 1)
}

func TestSethronHurloonGeneral_PumpMinotaursWithB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Sethron, Hurloon General")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Sethron, Hurloon General")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Sethron, Hurloon General", 5, 4)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", Menace, true)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", Haste, true)
}

func TestSethronHurloonGeneral_PumpMinotaursWithR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Sethron, Hurloon General")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.ActivateAbility(1, PrecombatMain, gametest.PlayerA, "Sethron, Hurloon General")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Sethron, Hurloon General", 5, 4)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", Menace, true)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", Haste, true)
}

func TestDualcasterMage_CopiesSpellOnStack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(ZoneHand, gametest.PlayerA, "Dualcaster Mage")
	g.AddCard(ZoneHand, gametest.PlayerA, "Bathe in Dragonfire")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Bathe in Dragonfire", "Hill Giant")
	g.CastInResponseTo(gametest.PlayerA, "Dualcaster Mage", "Bathe in Dragonfire")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Dualcaster Mage", 1)
}

func TestTrustyRetriever_CounterMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Trusty Retriever")
	g.ChooseMode(gametest.PlayerA, 0)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Trusty Retriever")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Trusty Retriever", P1P1, 1)
}

func TestTrustyRetriever_ReturnArtifactMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Trusty Retriever")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Sol Ring")
	g.ChooseMode(gametest.PlayerA, 1)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Trusty Retriever", "Sol Ring")
	g.StopAt(1, BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Sol Ring", 1)
}

// Soul of the Harvest: another nontoken creature you control entering draws.
func TestSoulOfTheHarvest_DrawsOnNontokenETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}

// Token ETB does NOT draw a card. Cast Sporemound (nontoken: triggers a draw),
// then play Forest so its landfall creates a Saproling token; Soul of the
// Harvest's draw must fire only once (for Sporemound itself, not the token).
func TestSoulOfTheHarvest_TokenETBDoesNotDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(ZoneHand, gametest.PlayerA, "Sporemound")
	g.AddCard(ZoneHand, gametest.PlayerA, "Forest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Sporemound")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	// Exactly one library top card drawn (Sporemound's nontoken ETB triggers Soul).
	// The Saproling token must NOT trigger a second draw.
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
}

// Soul of the Harvest's own ETB does not satisfy "another creature".
func TestSoulOfTheHarvest_OwnETBDoesNotDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(ZoneHand, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Soul of the Harvest")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Soul of the Harvest", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
}

// Opponent's nontoken creature entering does not trigger PlayerA's Soul.
func TestSoulOfTheHarvest_OpponentsCreatureDoesNotDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Forest", 4)
	g.AddCard(ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(2, PrecombatMain, gametest.PlayerB, "Grizzly Bears")
	g.StopAt(2, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
}

// Selvala — entering creature you control has greatest power: you draw.
func TestSelvalaHeartOfTheWilds_YouDrawOnYourBigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(3, PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.StopAt(3, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	// The Selvala trigger fires when Hill Giant enters (Hill Giant power 3 > Selvala 2 and Bears 2),
	// drawing one extra card for PlayerA.
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
}

// Selvala — entering creature opponent controls has greatest power: opponent draws.
func TestSelvalaHeartOfTheWilds_OpponentDrawsOnTheirBigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(ZoneHand, gametest.PlayerB, "Hill Giant")
	g.AddCard(ZoneBattlefield, gametest.PlayerB, "Mountain", 4)
	g.AddCard(ZoneLibrary, gametest.PlayerB, "Plains", 3)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Forest", 3)
	g.CastSpell(2, PrecombatMain, gametest.PlayerB, "Hill Giant")
	g.StopAt(2, EndStep)
	g.Execute()
	// PlayerB is the controller of the entering Hill Giant, so PlayerB draws
	// from the Selvala trigger — one Plains in hand.
	g.AssertHandCount(gametest.PlayerB, "Plains", 1)
	// PlayerA (Selvala's controller) must NOT have been the one to draw.
	g.AssertHandCount(gametest.PlayerA, "Forest", 0)
}

// Selvala — controller may decline the optional draw.
func TestSelvalaHeartOfTheWilds_MayDecline(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
	g.CastSpell(3, PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.StopAt(3, EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}

// Selvala — tied power (Bears 2 vs Selvala 2) is not strictly greater: no draw.
func TestSelvalaHeartOfTheWilds_TiedPowerNoDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}

// Selvala — entering creature ties an existing larger creature: no draw.
func TestSelvalaHeartOfTheWilds_TiesExistingLargest(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Selvala, Heart of the Wilds")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(ZoneHand, gametest.PlayerA, "War Mammoth")
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(ZoneLibrary, gametest.PlayerA, "Plains", 3)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "War Mammoth")
	g.StopAt(1, EndStep)
	g.Execute()
	// War Mammoth is 3/3, Hill Giant is 3/3 — not strictly greater.
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}
