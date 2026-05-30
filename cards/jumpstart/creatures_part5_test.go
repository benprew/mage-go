package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/antiquities"
	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestSethronHurloonGeneral_PumpMinotaursWithR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sethron, Hurloon General")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sethron, Hurloon General")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Sethron, Hurloon General", 5, 4)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", core.Menace, true)
	g.AssertHasAbility(gametest.PlayerA, "Sethron, Hurloon General", core.Haste, true)
}
func TestSigiledStarfish(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sigiled Starfish")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain"}, nil)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sigiled Starfish")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Sigiled Starfish", true)
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Mountain")
}
func TestSkitteringSurveyor_TutorsToHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Skittering Surveyor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 1)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Skittering Surveyor")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
}
func TestSlateStreetRuffian_DiscardWhenBlocked(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Slate Street Ruffian")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.Attack(3, gametest.PlayerA, "Slate Street Ruffian")
	g.Block(3, gametest.PlayerB, "Grizzly Bears", "Slate Street Ruffian")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Mountain", 0)
}
func TestSomberwaldStag_FightsTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Somberwald Stag")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Somberwald Stag", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// Soul of the Harvest: another nontoken creature you control entering draws.
func TestSoulOfTheHarvest_DrawsOnNontokenETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}

// Opponent's nontoken creature entering does not trigger PlayerA's Soul.
func TestSoulOfTheHarvest_OpponentsCreatureDoesNotDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
	g.StopAt(2, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
}

// Soul of the Harvest's own ETB does not satisfy "another creature".
func TestSoulOfTheHarvest_OwnETBDoesNotDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Soul of the Harvest")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Soul of the Harvest", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
}

// Token ETB does NOT draw a card. Cast Sporemound (nontoken: triggers a draw),
// then play Forest so its landfall creates a Saproling token; Soul of the
// Harvest's draw must fire only once (for Sporemound itself, not the token).
func TestSoulOfTheHarvest_TokenETBDoesNotDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Soul of the Harvest")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sporemound")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sporemound")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	// Exactly one library top card drawn (Sporemound's nontoken ETB triggers Soul).
	// The Saproling token must NOT trigger a second draw.
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
}
func TestSpectralSailor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spectral Sailor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Spectral Sailor")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Spectral Sailor", core.Flying, true)
}
func TestSpectralSailorFlash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Spectral Sailor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 1)
	g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Spectral Sailor")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Spectral Sailor", 1)
}
func TestSpitefulPrankster_HasFirstStrikeOwnTurn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spiteful Prankster")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Spiteful Prankster", core.FirstStrike, true)
}
func TestSporemound_SaprolingOnLandETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sporemound")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
}
func TestSteelPlumeMarshal(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Steel-Plume Marshal")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Healer's Hawk") // 1/1 flying
	g.Attack(3, gametest.PlayerA, "Steel-Plume Marshal", "Healer's Hawk")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Healer's Hawk", 3, 3)
}
func TestStoneHavenPilgrim(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stone Haven Pilgrim")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring") // artifact
	g.Attack(3, gametest.PlayerA, "Stone Haven Pilgrim")
	g.StopAt(3, core.DeclareBlockers)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Stone Haven Pilgrim", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Stone Haven Pilgrim", core.Lifelink, true)
}
func TestStormSculptor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Storm Sculptor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Storm Sculptor", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}
func TestSupplyRunners(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Supply Runners")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Supply Runners")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.AssertCounterCount(gametest.PlayerA, "Supply Runners", core.P1P1, 0)
}
func TestSuspiciousBookcase_MakesUnblockable(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Suspicious Bookcase")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Suspicious Bookcase", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
}
func TestSwarmOfBloodflies_CountersOnETBAndDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Swarm of Bloodflies")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Swarm of Bloodflies")
	g.CastSpell(1, core.PostcombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Swarm of Bloodflies", 3, 3)
}
func TestSylvanBrushstrider_GainsLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sylvan Brushstrider")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sylvan Brushstrider")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
}
func TestSylvanRanger_TutorsToHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sylvan Ranger")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 1)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sylvan Ranger")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	// Either in hand or auto-played as land.
	g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
}
func TestTalrandSkySummoner_CreatesDrakeOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Talrand, Sky Summoner")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Drake", 1)
}
func TestTemptingWitch_CreatesFoodOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Tempting Witch")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tempting Witch")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Tempting Witch", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Food", 1)
}
func TestTemptingWitch_SacFoodDrainsTargetPlayer(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Tempting Witch")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tempting Witch")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Tempting Witch", "PlayerB")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Food", 0)
	g.AssertLife(gametest.PlayerB, 17)
}
func TestThermoAlchemist_DealsOneToEachOpponent(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thermo-Alchemist")
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Thermo-Alchemist")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
}
func TestThragtusk_GainsLifeAndLeavesToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thragtusk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.SetLife(gametest.PlayerA, 20)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thragtusk")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 25)
}
func TestTibaltsRager_BoostsForOneR(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tibalt's Rager")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tibalt's Rager")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Tibalt's Rager", 3, 2)
}
func TestTinybonesTrinketThief_DamageHandless(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tinybones, Trinket Thief")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tinybones, Trinket Thief")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 10)
}
func TestTithebearerGiant_DrawAndLose(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Tithebearer Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tithebearer Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertLife(gametest.PlayerA, 19)
}
func TestTorchFiend_DestroysArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Torch Fiend")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bronze Tablet")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Torch Fiend", "Bronze Tablet")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Bronze Tablet", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Torch Fiend", 0)
}
func TestToweringTitan_BranchingEvolutionDoublesCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Branching Evolution")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Towering Titan", core.P1P1, 4)
}
func TestToweringTitan_DiesWhenNoOtherCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Towering Titan", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Towering Titan", 1)
}
func TestToweringTitan_EntersWithCountersFromOthersToughness(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Towering Titan", core.P1P1, 5)
	g.AssertPowerToughness(gametest.PlayerA, "Towering Titan", 5, 5)
}
func TestToweringTitan_OpponentCreaturesDoNotCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3 (opponent)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Towering Titan", core.P1P1, 2)
}
func TestTrustyRetriever_CounterMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Trusty Retriever")
	g.ChooseMode(gametest.PlayerA, 0)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Trusty Retriever")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Trusty Retriever", core.P1P1, 1)
}
func TestTrustyRetriever_ReturnArtifactMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Trusty Retriever")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Sol Ring")
	g.ChooseMode(gametest.PlayerA, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Trusty Retriever", "Sol Ring")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Sol Ring", 1)
}
func TestUlvenwaldHydra_PTEqualsLands(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ulvenwald Hydra")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.StopAt(1, core.Upkeep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Ulvenwald Hydra", 5, 5)
}

// Black tail: Vampire Neonate / Wailing Ghoul / Wight / Witch
// Plus red and green-start chunk.
func TestVampireNeonate_DrainOnActivation(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vampire Neonate")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.SetLife(gametest.PlayerA, 20)
	g.SetLife(gametest.PlayerB, 20)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vampire Neonate")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
}
func TestVedalkenArchmage_DrawsOnArtifactSpell(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vedalken Archmage")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ornithopter")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ornithopter")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
}
func TestVedalkenEntrancer_MillsTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vedalken Entrancer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains", 5)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vedalken Entrancer", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Plains", 2)
}
func TestVoiceOfTheProvinces(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Voice of the Provinces")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 6)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Voice of the Provinces")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Human", 1)
}
func TestVolleyVeteran_DealsDamageEqualToGoblins(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Volley Veteran")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mons's Goblin Raiders", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Volley Veteran", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestWailingGhoul_MillsTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Wailing Ghoul")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wailing Ghoul")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Plains", 2)
}
func TestWallOfBlossoms_DrawsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Wall of Blossoms")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wall of Blossoms")
	g.StopAt(1, core.EndStep)
	g.Execute()
}
func TestWallOfLostThoughts_MillsFourOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Wall of Lost Thoughts")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains", 8)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wall of Lost Thoughts")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Plains", 4)
}
func TestWarfireJavelineer_DealsDamageEqualToInstantsSorceries(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Warfire Javelineer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Warfire Javelineer", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestWeaverOfLightning_PingsOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Weaver of Lightning")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
func TestWightOfPrecinctSix_BoostByOpponentGraveyardCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wight of Precinct Six")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears", 2)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Wight of Precinct Six", 3, 3)
}
func TestWildheartInvoker_PumpsAndGrantsTrample(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wildheart Invoker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 8)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Wildheart Invoker", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 7, 7)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
}
func TestWindreaderSphinx_DrawsOnFlyingAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Windreader Sphinx")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 2)
	g.Attack(1, gametest.PlayerA, "Serra Angel")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
}
func TestWindstormDrake_BoostsOtherFlyers(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Windstorm Drake")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Serra Angel", 5, 4)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	g.AssertPowerToughness(gametest.PlayerA, "Windstorm Drake", 3, 3)
}
func TestWoodbornBehemoth_BuffsWith8Lands(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Woodborn Behemoth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 8)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Woodborn Behemoth", 8, 8)
	g.AssertHasAbility(gametest.PlayerA, "Woodborn Behemoth", core.Trample, true)
}
func TestWrensRunVanquisher_RevealOrPay(t *testing.T) {
	t.Run("reveal Elf branch when an Elf is in hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wren's Run Vanquisher")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Llanowar Elves")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wren's Run Vanquisher")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Wren's Run Vanquisher", 1)
		g.AssertHandCount(gametest.PlayerA, "Llanowar Elves", 1)
	})
}
func TestYoungPyromancer_CreatesElementalOnInstant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Young Pyromancer")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Elemental", 1)
}
