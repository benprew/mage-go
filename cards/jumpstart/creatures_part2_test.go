package jumpstart

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/antiquities"
	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

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

func TestBloodArtist_DrainsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blood Artist")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
}

func TestBloodhunterBat_DrainsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bloodhunter Bat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bloodhunter Bat")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertLife(gametest.PlayerB, 18)
}

func TestBlackCat_RandomDiscardOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Cat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Plains")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.Block(2, gametest.PlayerA, "Black Cat", "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Plains", 1)
}

func TestBlightedBat_GainsHaste(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Blighted Bat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blighted Bat")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Blighted Bat")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Blighted Bat", core.Haste, true)
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

func TestLilianasElite_PTByGraveyardCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Liliana's Elite")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Plains")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Liliana's Elite", 4, 4)
}

func TestGristleGrinner_BoostOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gristle Grinner")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Gristle Grinner", 5, 5)
}

func TestMiasmicMummy_EachPlayerDiscards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Miasmic Mummy")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Miasmic Mummy")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Plains", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
}

func TestGiftedAetherborn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gifted Aetherborn")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Gifted Aetherborn", core.Deathtouch, true)
	g.AssertHasAbility(gametest.PlayerA, "Gifted Aetherborn", core.Lifelink, true)
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

func TestBurglarRat_OpponentDiscards(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Burglar Rat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Burglar Rat")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
}

func TestCadaverImp_ReturnsCreatureFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cadaver Imp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cadaver Imp")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestCrowOfDarkTidings_MillsOnETBAndDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Crow of Dark Tidings")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 6)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Crow of Dark Tidings")
	g.CastSpell(1, core.PostcombatMain, gametest.PlayerA, "Lightning Bolt", "Crow of Dark Tidings")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 4)
}

func TestFalkenrathNoble_DrainsOnAnyDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Falkenrath Noble")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
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

func TestCorpseHauler_SacToReturn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Corpse Hauler")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Corpse Hauler", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Corpse Hauler", 1)
}

func TestDranaLiberatorOfMalakir_BoostsAttackersOnHit(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drana, Liberator of Malakir")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(3, gametest.PlayerA, "Drana, Liberator of Malakir", "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Drana, Liberator of Malakir", 3, 4)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}

func TestDutifulAttendant_ReturnsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dutiful Attendant")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Dutiful Attendant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestFesteringNewt_DebuffOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Festering Newt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Festering Newt")
	g.ChoosePermanent(gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 2, 2)
}

func TestGhoulraiser_ReturnsZombieFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ghoulraiser")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Black Cat")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ghoulraiser")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Black Cat", 1)
}

func TestGravewaker_ReanimatesTapped(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gravewaker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 7)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Gravewaker", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
}

func TestHarvesterOfSouls_DrawsOnNontokenDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Harvester of Souls")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
}

func TestBloodHost_SacForCounterAndLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blood Host")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Blood Host")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertPowerToughness(gametest.PlayerA, "Blood Host", 4, 4)
}

func TestKelsFightFixer_SacForIndestructible(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kels, Fight Fixer")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Kels, Fight Fixer", core.Indestructible, true)
}

func TestLilianasReaver_DiscardAndZombieToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Liliana's Reaver")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain")
	g.Attack(3, gametest.PlayerA, "Liliana's Reaver")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
	g.AssertHandCount(gametest.PlayerB, "Mountain", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Zombie", 1)
}

func TestMireTriton_MillsAndGains(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mire Triton")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mox Ruby", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mire Triton")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 2)
	g.AssertLife(gametest.PlayerA, 22)
}

func TestNightshadeStinger_CantBlock(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nightshade Stinger")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.Block(2, gametest.PlayerA, "Nightshade Stinger", "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 17)
	g.AssertPermanentCount(gametest.PlayerA, "Nightshade Stinger", 1)
}

func TestNocturnalFeeder_DrainsOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nocturnal Feeder")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Nocturnal Feeder")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 22)
	g.AssertLife(gametest.PlayerB, 18)
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

func TestTinybonesTrinketThief_DamageHandless(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tinybones, Trinket Thief")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tinybones, Trinket Thief")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 10)
}

func TestLawlessBroker_CounterOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lawless Broker")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Lawless Broker")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}

func TestCauldronFamiliar_DrainsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cauldron Familiar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cauldron Familiar")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertPermanentCount(gametest.PlayerA, "Cauldron Familiar", 1)
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

func TestBloodbondVampire_GainsCounterOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bloodbond Vampire")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertCounterCount(gametest.PlayerA, "Bloodbond Vampire", core.P1P1, 1)
}

func TestBloodbondVampire_NoCounterOnOpponentLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bloodbond Vampire")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
	g.CastSpell(1, core.PostcombatMain, gametest.PlayerB, "Healing Salve", "PlayerB")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Bloodbond Vampire", core.P1P1, 0)
}

func TestKalastriaNightwatch_GainsFlyingOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kalastria Nightwatch")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AssertHasAbility(gametest.PlayerA, "Kalastria Nightwatch", core.Flying, false)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Kalastria Nightwatch", core.Flying, true)
}

func TestMalakirFamiliar_PumpsOnLifeGain(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Malakir Familiar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Malakir Familiar", 3, 2)
}

func TestFellSpecter_DiscardCausesLifeLoss(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fell Specter")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Twist")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mind Twist", 1, "PlayerB")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Fell Specter's ETB ("target opponent discards a card") fires when added to
	// the battlefield, then Mind Twist X=1 discards another. Two discards → two
	// trigger fires of "Whenever an opponent discards a card, that player loses 2
	// life" (CR 701.8: each card discarded is its own event) → 4 life lost.
	g.AssertLife(gametest.PlayerB, 16)
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

func TestKelsFightFixer_DrawOnSacrifice(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// One Swamp pays the {1} sacrifice cost, another pays {U/B} for the may-pay draw.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kels, Fight Fixer")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}

func TestKelsFightFixer_DeclineDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kels, Fight Fixer")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kels, Fight Fixer")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
}

func TestDrainpipeVermin_DiscardOnDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drainpipe Vermin")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Drainpipe Vermin")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestEternalTaskmaster_ReturnOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Eternal Taskmaster")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.Attack(3, gametest.PlayerA, "Eternal Taskmaster")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}


func TestBelltowerSphinx_DamagerControllerMills(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Belltower Sphinx")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Hill Giant", 5)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Belltower Sphinx")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 3)
}

func TestMausoleumTurnkey_OpponentChoosesReturn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mausoleum Turnkey")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerB, "Hill Giant")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mausoleum Turnkey")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}
