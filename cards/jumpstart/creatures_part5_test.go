package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Kira, Great Glass-Spinner: targeting Kira herself counters the spell, since
// the granted "creatures you control" trigger now includes the source.
func TestKiraGreatGlassSpinner_CountersTargetingSelf(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kira, Great Glass-Spinner")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Kira, Great Glass-Spinner")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Kira, Great Glass-Spinner", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
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
