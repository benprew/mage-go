package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited" // register base cards (basic lands, vanilla creatures)
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestAegisOfTheHeavens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Aegis of the Heavens")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Aegis of the Heavens", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 9)
}

func TestAggressiveUrge(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Aggressive Urge")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Aggressive Urge", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
}

func TestAgonizingSyphon(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 17)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Agonizing Syphon")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Agonizing Syphon", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 20)
	g.AssertLife(gametest.PlayerB, 17)
}

func TestAngelicEdict(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Angelic Edict")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Angelic Edict", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertExileCount("Grizzly Bears", 1)
}

func TestArborArmament(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Arbor Armament")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Arbor Armament", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Reach, true)
}

func TestBoneSplinters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bone Splinters")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bone Splinters", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestDealDamageDouseInGloom(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 17)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Douse in Gloom")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Douse in Gloom", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 19)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestDragonFodderTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Dragon Fodder")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dragon Fodder")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 2)
}

func TestRaiseTheAlarmTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Raise the Alarm")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Raise the Alarm")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Soldier", 2)
}

func TestBatheInDragonfire(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bathe in Dragonfire")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bathe in Dragonfire", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestLastGasp(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Last Gasp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Last Gasp", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestLanguishWipes(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Languish")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Languish")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestInspiredCharge(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Inspired Charge")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Inspired Charge")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 3)
}

func TestExclude(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Exclude")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
	g.CastInResponseTo(gametest.PlayerA, "Exclude", "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestThoughtScour(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thought Scour")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thought Scour", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Plains", 2)
}

func TestReanimate(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Reanimate")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Reanimate", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerA, 16)
}

func TestActOfTreason(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Act of Treason")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Act of Treason", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Haste, true)
}

func TestInnocentBlood(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Innocent Blood")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Innocent Blood")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestSpittingEarth(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Spitting Earth")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Spitting Earth", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestOutnumberCountsCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Outnumber")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Outnumber", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestAerialAssault(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mesa Pegasus")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Aerial Assault")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Aerial Assault", "Hill Giant")
	g.StopAt(3, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerA, 18)
}

func TestAssassinsStrike(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Assassin's Strike")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Assassin's Strike", "Hill Giant")
	g.ChooseDiscard(gametest.PlayerB, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestAugerSpree(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Auger Spree")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Auger Spree", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestBarterInBlood(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Barter in Blood")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Barter in Blood")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
}

func TestBattlefieldPromotion(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 18)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Battlefield Promotion")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Battlefield Promotion", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 20)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.FirstStrike, true)
}

func TestBefuddle(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Befuddle")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Befuddle", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", -1, 3)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestBlindblast(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Blindblast")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blindblast", "Mesa Pegasus")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestBloodDivination(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Blood Divination")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blood Divination")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 3)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
}

func TestCemeteryRecruitment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Scathe Zombies")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cemetery Recruitment")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cemetery Recruitment", "Scathe Zombies")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Scathe Zombies", 1)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}

func TestChartACourse(t *testing.T) {
	t.Run("draw two and discard if not attacked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chart a Course")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant", 3)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chart a Course")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	})
	t.Run("no discard if attacked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chart a Course")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant", 3)
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(3, core.PostcombatMain, gametest.PlayerA, "Chart a Course")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 3)
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
	})
}

func TestCloudshift(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Cloudshift")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cloudshift", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestCollateralDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Collateral Damage")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Collateral Damage", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestDivineArrow(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Divine Arrow")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.CastSpell(2, core.DeclareAttackers, gametest.PlayerA, "Divine Arrow", "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestEssenceFlux(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Essence Flux")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Essence Flux", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestFlameLash(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Flame Lash")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flame Lash", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
}

func TestFlamesOfTheRazeBoar(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Air Elemental")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Flames of the Raze-Boar")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flames of the Raze-Boar", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestFlurryOfHornsTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Flurry of Horns")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flurry of Horns")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Minotaur", 2)
}

func TestFuneralRites(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Funeral Rites")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Funeral Rites")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 2)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 2)
}

func TestGoblinRallyTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Goblin Rally")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Goblin Rally")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Goblin", 4)
}

func TestVolcanicFalloutDamagesAll(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Volcanic Fallout")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Volcanic Fallout")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertLife(gametest.PlayerB, 18)
}
