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
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 2)
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
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wall of Lost Thoughts", "PlayerB")
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
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(1, gametest.PlayerB, "Hill Giant")
	g.Block(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
	g.AssertLife(gametest.PlayerB, 19)
}

func TestBloodhunterBat_DrainsOnETB(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bloodhunter Bat")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bloodhunter Bat", "PlayerB")
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
	g.AssertHandCount(gametest.PlayerA, "Mox Ruby", 2)
	g.AssertLife(gametest.PlayerA, 19)
}

func TestPhyrexianGargantua_DrawTwoLoseTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Phyrexian Gargantua")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 2)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Gargantua")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 2)
	g.AssertLife(gametest.PlayerA, 18)
}

func TestTithebearerGiant_DrawAndLose(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Tithebearer Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 6)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tithebearer Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
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
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(1, gametest.PlayerB, "Hill Giant")
	g.Block(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
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
