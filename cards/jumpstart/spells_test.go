package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited" // register base cards (basic lands, vanilla creatures)
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
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
	t.Run("from your graveyard", func(t *testing.T) {
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
	})

	t.Run("from opponent's graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Reanimate")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Reanimate", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertLife(gametest.PlayerA, 16)
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 0)
	})
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

func TestHeartfire(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Heartfire")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Heartfire", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestHomingLightning(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Homing Lightning")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Homing Lightning", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
}

func TestImmolatingGyre(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 6)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Bathe in Dragonfire", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Immolating Gyre")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Immolating Gyre")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

func TestInspiringCall(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Inspiring Call")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Inspiring Call")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Indestructible, true)
}

func TestLaunchParty(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Launch Party")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Launch Party", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestLeaveInTheDust(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Leave in the Dust")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Leave in the Dust", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestLifecraftersGift(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant", core.P1P1, 1)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lifecrafter's Gift")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lifecrafter's Gift", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
	g.AssertCounterCount(gametest.PlayerA, "Hill Giant", core.P1P1, 2)
}

func TestMagmaquake(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Magmaquake")
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Magmaquake", 3)
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Mesa Pegasus", 1)
}

func TestMomentOfHeroism(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Moment of Heroism")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Moment of Heroism", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, true)
}

func TestMugging(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bog Wraith")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mugging")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mugging", "Bog Wraith")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.Block(1, gametest.PlayerB, "Bog Wraith", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
}

func TestReleaseTheDogsTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Release the Dogs")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Release the Dogs")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Dog", 4)
}

func TestRiseOfTheDarkRealms(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 9)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Rise of the Dark Realms")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rise of the Dark Realms")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
}

func TestSarkhansRageNoDragons(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sarkhan's Rage")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sarkhan's Rage", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 15)
	g.AssertLife(gametest.PlayerA, 18)
}

func TestTakeHeart(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Take Heart")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.DeclareAttackers, gametest.PlayerA, "Take Heart", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
}

func TestTalrandsInvocationTokens(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Talrand's Invocation")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Talrand's Invocation")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Drake", 2)
}

func TestThoughtCollapse(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thought Collapse")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Hill Giant", 5)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
	g.CastInResponseTo(gametest.PlayerA, "Thought Collapse", "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 3)
}

func TestWildsize(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Wildsize")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wildsize", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
}

func TestWhelmingWave(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Whelming Wave")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Whelming Wave")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerB, "Hill Giant", 1)
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

func TestMagmaJet(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Magma Jet")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.ChooseScry(gametest.PlayerA, []string{"Forest", "Island"}, nil)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Magma Jet", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertLibraryTop(gametest.PlayerA, "Plains", "Forest", "Island")
}

func TestVoyagesEnd(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Voyage's End")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain"}, nil)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Voyage's End", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Mountain")
}

func TestRiddleOfLightningScry(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Riddle of Lightning")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Swamp")
	g.ChooseScry(gametest.PlayerA, []string{"Forest", "Island", "Plains"}, nil)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Riddle of Lightning")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Swamp", "Forest", "Island", "Plains")
}

func TestBakeIntoAPie_DestroysAndCreatesFood(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bake into a Pie")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bake into a Pie", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertPermanentCount(gametest.PlayerA, "Food", 1)
}

func TestDauntlessOnslaught(t *testing.T) {
	t.Run("two targets each get +2/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dauntless Onslaught")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dauntless Onslaught", "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 5, 5)
	})
	t.Run("single target gets +2/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dauntless Onslaught")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dauntless Onslaught", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3)
	})
	t.Run("zero targets resolves with no effect", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dauntless Onslaught")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dauntless Onslaught")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
		g.AssertGraveyardCount(gametest.PlayerA, "Dauntless Onslaught", 1)
	})
}

func TestGirdForBattle(t *testing.T) {
	t.Run("two targets each get a +1/+1 counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 1)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Gird for Battle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Gird for Battle", "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 4, 4)
	})
	t.Run("zero targets is legal", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 1)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Gird for Battle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Gird for Battle")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Gird for Battle", 1)
	})
}

func TestTandemTactics(t *testing.T) {
	t.Run("two targets each get +1/+2 and you gain 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 18)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tandem Tactics")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tandem Tactics", "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 4)
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 4, 5)
		g.AssertLife(gametest.PlayerA, 20)
	})
	t.Run("zero targets still gains 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 18)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tandem Tactics")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tandem Tactics")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestFlamesOfTheFirebrand(t *testing.T) {
	t.Run("split 1/2 between two creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flames of the Firebrand")
		g.ChooseDamageDistribution(gametest.PlayerA, map[string]int{
			"Grizzly Bears": 1,
			"Hill Giant":    2,
		})
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flames of the Firebrand", "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})
	t.Run("3 to one creature kills it", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flames of the Firebrand")
		g.ChooseDamageDistribution(gametest.PlayerA, map[string]int{"Grizzly Bears": 3})
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flames of the Firebrand", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
	t.Run("split between player and creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flames of the Firebrand")
		g.ChooseDamageDistribution(gametest.PlayerA, map[string]int{
			"PlayerB":       2,
			"Grizzly Bears": 1,
		})
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flames of the Firebrand", "PlayerB", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestHungryFlames(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hungry Flames")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hungry Flames", "Hill Giant", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertLife(gametest.PlayerB, 18)
}

func TestPeelFromReality(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Peel from Reality")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Peel from Reality", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerB, "Hill Giant", 1)
}

func TestNaturesWay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Nature's Way")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Nature's Way", "Hill Giant", "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertHasAbility(gametest.PlayerA, "Hill Giant", core.Vigilance, true)
	g.AssertHasAbility(gametest.PlayerA, "Hill Giant", core.Trample, true)
}

func TestCrushingCanopy_DestroyFlying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mesa Pegasus")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Crushing Canopy")
	g.ChooseMode(gametest.PlayerA, 0)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Crushing Canopy", "Mesa Pegasus")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mesa Pegasus", 1)
}

func TestCrushingCanopy_DestroyEnchantment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Pacifism")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Crushing Canopy")
	g.ChooseMode(gametest.PlayerA, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Crushing Canopy", "Pacifism")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Pacifism", 1)
}

func TestFortify_PowerMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fortify")
	g.ChooseMode(gametest.PlayerA, 0)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fortify")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 2)
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 3, 3)
}

func TestFortify_ToughnessMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fortify")
	g.ChooseMode(gametest.PlayerA, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fortify")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 4)
}

func TestValorousStance_Indestructible(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Valorous Stance")
	g.ChooseMode(gametest.PlayerA, 0)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Valorous Stance", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Indestructible, true)
}

func TestValorousStance_DestroyTough(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Valorous Stance")
	g.ChooseMode(gametest.PlayerA, 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Valorous Stance", "Serra Angel")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Serra Angel", 1)
}

func TestDoublecast_CopiesNextInstantOrSorcery(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Doublecast")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bathe in Dragonfire")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Doublecast")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bathe in Dragonfire", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Bathe deals 4; copy also resolves (same target by default → second resolution sees creature gone, copy fizzles).
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// Lightning Axe: "As an additional cost to cast this spell, discard a card or
// pay {5}." With both options payable, default ChooseMode = 0 → discard branch.
func TestLightningAxe_DiscardBranch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Axe")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.ChooseDiscard(gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Axe", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	// Discard branch consumed the Plains from hand.
	g.AssertGraveyardCount(gametest.PlayerA, "Plains", 1)
	g.AssertHandCount(gametest.PlayerA, "Plains", 0)
}

// Lightning Axe: with no other cards in hand at cast time, discard branch is
// unpayable and EitherCost auto-routes to the {5} mana branch.
func TestLightningAxe_PayFiveBranch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 6)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Axe")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Axe", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

// Pillar of Flame: 2 damage to a 2/2 creature kills it; the corpse goes to
// exile (not the graveyard) per "if it would die this turn, exile it instead".
func TestPillarOfFlame_ExilesKilledCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Pillar of Flame")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pillar of Flame", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
}

// Pillar of Flame: dealing 2 to a 3/3 doesn't kill it, so the exile-replacement
// is irrelevant — the creature stays on the battlefield.
func TestPillarOfFlame_NonLethalDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Pillar of Flame")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pillar of Flame", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
}

// Exhume: each player picks a creature card from their own graveyard and
// puts it onto the battlefield. With one creature in each graveyard, both
// come back.
func TestExhume_BothPlayersReanimate(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Exhume")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Exhume")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// Exhume: a player with no creature card in their graveyard simply skips.
func TestExhume_OneEmptyGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Exhume")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Exhume")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
}

// Sweep Away: non-attacking creature is bounced to its owner's hand.
func TestSweepAway_NonAttacking_BouncesToHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sweep Away")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sweep Away", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// Sweep Away: attacking creature with controller saying "yes" goes on top of
// owner's library instead of returning to hand.
func TestSweepAway_Attacking_PutOnTopOfLibrary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Sweep Away")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(true)
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.DeclareAttackers, gametest.PlayerB, "Sweep Away", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears")
}

// Sweep Away: attacking creature, but controller declines top-of-library —
// falls back to plain bounce.
func TestSweepAway_Attacking_DeclinesTop(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Sweep Away")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.DeclareAttackers, gametest.PlayerB, "Sweep Away", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// Pillar of Flame: targeting a player just deals 2 damage; no exile replacement.
func TestPillarOfFlame_PlayerTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Pillar of Flame")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pillar of Flame", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
}

func TestHuntersInsight_DrawsOnEachCombatHit(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hunter's Insight")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hunter's Insight", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Grizzly Bears (2 power) connects on turn 1: trigger fires, draw 2 cards.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 2)
}

func TestHuntersInsight_ExpiresEndOfTurn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hunter's Insight")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hunter's Insight", "Grizzly Bears")
	// Don't attack on turn 1; trigger expires at end of turn 1.
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// Without expiration, Bears would draw 2 extra cards from the trigger; but
	// it expired at end of turn 1. PlayerA only drew 1 normal draw-step card on
	// turn 3 (PlayerB's turn 2 draw doesn't touch PlayerA's library), so
	// library went from 5 → 4.
	g.AssertLibraryCount(gametest.PlayerA, "Lightning Bolt", 4)
}

func TestPathToExile_ExilesAndOpponentMaySearch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Mountain", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Path to Exile")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ChooseMode(gametest.PlayerB, 0)
	g.ChooseFromLibrary(gametest.PlayerB, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Path to Exile", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Forest", 1)
	g.AssertTapped(gametest.PlayerB, "Forest", true)
}

// Commune with Dinosaurs: scripted bottom order is honored, and the chosen
// Dinosaur card is moved to hand. The library after resolution should be the
// pre-existing tail (cards 6..N) followed by the four un-picked top cards in
// the order the player scripted (last name = new bottom card).
func TestCommuneWithDinosaurs_BottomOrderHonored(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Commune with Dinosaurs")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	// Top 5 of library, in order:
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")         // top
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Orazca Frillback") // chosen Dinosaur
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Aegis of the Heavens")
	// Tail (untouched):
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")

	g.ChooseFromLibrary(gametest.PlayerA, "Orazca Frillback")
	// Place the four un-picked cards on bottom, deepest last:
	// shallow -> deep: Mountain, Plains, Lightning Bolt, Aegis of the Heavens.
	g.ChooseScry(gametest.PlayerA, []string{
		"Mountain", "Plains", "Lightning Bolt", "Aegis of the Heavens",
	}, nil)

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Commune with Dinosaurs")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertHandCount(gametest.PlayerA, "Orazca Frillback", 1)
	g.AssertLibraryCount(gametest.PlayerA, "Orazca Frillback", 0)
	// Tail (Swamp, Island) is now on top, then the chosen bottom order.
	g.AssertLibraryTop(gametest.PlayerA,
		"Swamp",
		"Island",
		"Mountain",
		"Plains",
		"Lightning Bolt",
		"Aegis of the Heavens",
	)
}

// Commune with Dinosaurs: when no revealed card is a Dinosaur or land, the
// player has no legal pick. All five cards are bottomed in chosen order and
// hand size is unchanged (other than the spell leaving hand for graveyard).
func TestCommuneWithDinosaurs_NoLegalPick(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Commune with Dinosaurs")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	// Top 5: all instants/sorceries (no creatures, no lands).
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Aegis of the Heavens")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Aggressive Urge")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Path to Exile")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Cloudshift")
	// Tail.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Swamp")

	g.ChooseScry(gametest.PlayerA, []string{
		"Cloudshift", "Path to Exile", "Aggressive Urge", "Aegis of the Heavens", "Lightning Bolt",
	}, nil)

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Commune with Dinosaurs")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// Nothing put into hand from the top 5.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 0)
	g.AssertHandCount(gametest.PlayerA, "Aegis of the Heavens", 0)
	g.AssertHandCount(gametest.PlayerA, "Aggressive Urge", 0)
	g.AssertHandCount(gametest.PlayerA, "Path to Exile", 0)
	g.AssertHandCount(gametest.PlayerA, "Cloudshift", 0)
	// All 5 are still in the library, on the bottom in the scripted order.
	g.AssertLibraryTop(gametest.PlayerA,
		"Swamp",
		"Cloudshift",
		"Path to Exile",
		"Aggressive Urge",
		"Aegis of the Heavens",
		"Lightning Bolt",
	)
}

// TestSavageStomp_CounterAndFight verifies +1/+1 counter then fight between
// two targets. A 2/2 (post-counter 3/3) fights a 3/3 — both die.
func TestSavageStomp_CounterAndFight(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Savage Stomp", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

// TestTimeToFeed_FightAndGain3 verifies the fight resolves and the on-death
// trigger fires for +3 life when the opponent's creature dies.
func TestTimeToFeed_FightAndGain3(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerA, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Time to Feed")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Time to Feed", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerA, 23)
}

// TestThirstForKnowledge_DiscardArtifact verifies the controller defaults to
// "yes pay" and discards an artifact card to avoid the 2-card discard.
func TestThirstForKnowledge_DiscardArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sol Ring")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Sol Ring", 1)
}

// TestThirstForKnowledge_PlayerChoosesArtifact verifies that with multiple
// artifact cards in hand, the controller picks which one to discard rather
// than the engine auto-picking the first.
func TestThirstForKnowledge_PlayerChoosesArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sol Ring")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mox Ruby")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.ChooseDiscard(gametest.PlayerA, "Mox Ruby")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Sol Ring", 0)
	g.AssertHandCount(gametest.PlayerA, "Sol Ring", 1)
}

// TestThirstForKnowledge_NoArtifactDiscardTwo verifies that when there is no
// artifact card in hand, the controller cannot pay the artifact branch and
// must discard two cards instead.
func TestThirstForKnowledge_NoArtifactDiscardTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Counterspell")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.ChooseDiscard(gametest.PlayerA, "Lightning Bolt", "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Counterspell", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Thirst for Knowledge", 1)
}

// TestThirstForKnowledge_DeclineArtifactBranch verifies that the controller
// can choose NOT to pay the discard-an-artifact branch even when an artifact
// is in hand, and instead discard two non-artifact cards of their choice.
func TestThirstForKnowledge_DeclineArtifactBranch(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mox Ruby")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Counterspell")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false)
	g.ChooseDiscard(gametest.PlayerA, "Lightning Bolt", "Counterspell")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Mox Ruby", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Counterspell", 1)
}

// TestReadTheRunes_X1Discard verifies that for X=1 with the controller
// declining the sacrifice option, the discard branch fires once.
func TestReadTheRunes_X1Discard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Read the Runes")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	tp := g.GetPlayer(gametest.PlayerA)
	tp.QueueMayAbilityChoices(false)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Read the Runes", 1)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Read the Runes", 1)
}

// TestDraconicRoar_RevealedDragonDealsExtra verifies the optional reveal pays
// off as +3 damage to the targeted creature's controller.
func TestDraconicRoar_RevealedDragonDealsExtra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertHandCount(gametest.PlayerA, "Shivan Dragon", 1)
}

// TestDraconicRoar_NoDragonNoBonus verifies that without a Dragon to reveal
// or control, only the base 3 damage applies and the controller takes no hit.
func TestDraconicRoar_NoDragonNoBonus(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerB, 20)
}

// TestDraconicRoar_ControlledDragonAtCastDealsExtra verifies that a Dragon
// the caster controls at cast time triggers the bonus damage on resolution.
// PlayerA declines the optional reveal — the controlled-Dragon half alone
// must satisfy the condition.
func TestDraconicRoar_ControlledDragonAtCastDealsExtra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Hatchling")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false) // decline optional reveal
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerB, 17)
}

// TestDraconicRoar_ControlledDragonLeavesBeforeResolution verifies CR 608.2g:
// the "controlled a Dragon as you cast this spell" condition is fixed at
// cast time, so a Dragon that is killed in response between cast and
// resolution still satisfies the condition. PlayerA's Dragon Hatchling
// (0/1) is killed by PlayerB's Lightning Bolt cast in response to
// Draconic Roar; when Draconic Roar resolves it must still deal the
// bonus damage to PlayerB.
func TestDraconicRoar_ControlledDragonLeavesBeforeResolution(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Hatchling")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false) // decline optional reveal — only the controlled half should matter
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.CastInResponseTo(gametest.PlayerB, "Lightning Bolt", "Dragon Hatchling")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Dragon Hatchling died to the Bolt before Draconic Roar resolved.
	g.AssertGraveyardCount(gametest.PlayerA, "Dragon Hatchling", 1)
	// Grizzly Bears died to Draconic Roar's 3 damage.
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	// Bonus damage from cast-time-snapshotted "controlled a Dragon" still applied.
	g.AssertLife(gametest.PlayerB, 17)
}

// TestDraconicRoar_RevealedDragonOnlyNoBattlefieldDragon verifies the
// reveal-half of the condition independently: PlayerA controls no Dragon
// but reveals a Dragon card from hand as the optional additional cost,
// triggering the bonus damage.
func TestDraconicRoar_RevealedDragonOnlyNoBattlefieldDragon(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertHandCount(gametest.PlayerA, "Shivan Dragon", 1)
	g.AssertLife(gametest.PlayerB, 17)
}

// TestExplore_AllowsSecondLand verifies that after Explore resolves, the
// active player's land-play allowance is bumped to 2. The harness's auto-
// land-play loop then plays both Forests in the same main phase.
func TestExplore_AllowsSecondLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Explore")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Explore")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Forest", 4)
}

// Long Road Home: exile target creature, return at next end step with a
// +1/+1 counter.
func TestLongRoadHome(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Long Road Home")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Long Road Home", "Grizzly Bears")
	g.StopAt(2, core.Upkeep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 3, 3)
}

// Elemental Uprising: target land becomes 4/4 Elemental w/ haste.
func TestElementalUprising_AnimatesLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Elemental Uprising")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Elemental Uprising", "Forest")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Forest", 4, 4)
}

// Riddle of Lightning: damage equals top card's mana value.
func TestRiddleOfLightning(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Riddle of Lightning")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	// Top card after Scry will still be Hill Giant; bottom-stuff with Mountains.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Riddle of Lightning", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Hill Giant CMC is 4.
	g.AssertLife(gametest.PlayerB, 16)
}

// Dance with Devils: creates two 1/1 Devil tokens with the dies trigger.
func TestDanceWithDevils_TokensWithDiesTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Dance with Devils")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dance with Devils")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Devil", 2)
}
