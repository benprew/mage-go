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

// Lawmage's Binding: enchanted creature's non-mana activated abilities can't be
// activated.
func TestLawmagesBinding_BlocksActivatedAbilities(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prodigal Sorcerer")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lawmage's Binding")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lawmage's Binding", "Prodigal Sorcerer")
	g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Prodigal Sorcerer", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// Activation blocked: PlayerA still at 20 life.
	g.AssertLife(gametest.PlayerA, 20)
}

// Assault Formation: a 0/4 Wall attacking deals damage equal to its toughness.
func TestAssaultFormation_AssignsToughnessAsDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Assault Formation")
	// Wall of Wood: 0/3 with defender. Use {G} to remove defender, then attack.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Assault Formation", "Wall of Wood")
	g.Attack(3, gametest.PlayerA, "Wall of Wood")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// 0/3 wall now assigns 3 damage instead of 0.
	g.AssertLife(gametest.PlayerB, 17)
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

// Terrarion: when the artifact is sacrificed (via its own activated ability),
// the LtB trigger draws a card.
func TestTerrarion_SacrificeDrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Terrarion")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Terrarion")
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Terrarion", 1)
	// Drew a Grizzly Bears (default agent doesn't auto-cast spells).
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}
