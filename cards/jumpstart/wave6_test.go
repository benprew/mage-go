package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

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

// Fell Specter: ETB makes target opponent discard.
func TestFellSpecter_ETBDiscard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fell Specter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Mountain", 3)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fell Specter")
	g.ChooseDiscard(gametest.PlayerB, "Mountain")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Mountain", 2)
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
}

// Goblin Goon: can't attack unless attacker controls strictly more
// creatures than the defending player.
func TestGoblinGoon_CantAttackEqualCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Goon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Goblin Goon")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 20)
}

// Goblin Goon: can't block unless its controller has strictly more
// creatures than attacking player.
func TestGoblinGoon_CantBlockEqualCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Goblin Goon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Goblin Goon", "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
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

// Keeper of Fables: non-Human combat damage triggers a draw.
func TestKeeperOfFables_NonHumanDrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Keeper of Fables")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 3)
	g.Attack(3, gametest.PlayerA, "Keeper of Fables")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// Keeper itself is a Cat — non-Human — so combat damage triggers draw.
	g.AssertHandCount(gametest.PlayerA, "Mountain", 1)
}
