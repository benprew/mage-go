package limited

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// =============================================================================
// Turn-structure tests — validate engine behavior against specific CR rules.
// Tests that expose engine gaps are skipped with t.Skip("engine gap: ...").
// Each test documents the CR rule and expected behavior.
// =============================================================================

// CR 500.5 — "When a step or phase ends, any unused mana left in a player's
// mana pool empties. This turn-based action doesn't use the stack."
func TestManaPoolEmptiesBetweenSteps(t *testing.T) {
	t.Skip("engine gap: mana pool does not empty between steps (CR 500.5)")
}

// CR 500.11 — "Some effects can give a player an extra turn, or skip a step,
// phase, or turn." This tests that a card like Time Walk grants an extra turn.
func TestExtraTurnEffects(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Time Walk")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Time Walk")
	g.StopAt(3, core.Upkeep)
	g.Execute()
	// After Time Walk, PlayerA takes turn 1 and an extra turn 2.
	// Turn 3 is PlayerB's turn. Verify we're on turn 3.
	g.AssertTotalTurns(3)
}

// CR 500.11 — Skip-step/phase/turn effects (e.g. "skip your draw step").
func TestSkipStepEffects(t *testing.T) {
	t.Skip("engine gap: no card tests skip-step effects yet (CR 500.11)")
}

// CR 500.12 — "Nothing observable can happen between steps within a single phase."
func TestNothingBetweenSteps(t *testing.T) {
	t.Skip("engine gap: no test for CR 500.12 observable-between-steps constraint")
}

// CR 503.2 — "If a spell or ability refers to an extra upkeep step, that step
// is created after the normal upkeep step." Tests cards that grant extra upkeep.
func TestExtraUpkeepStep(t *testing.T) {
	t.Skip("engine gap: extra upkeep step effects not implemented (CR 503.2)")
}

// CR 505.1a — "An ability or effect may state that a player skips the combat
// phase." Tests Mystic Decree or similar.
func TestSkipCombatPhase(t *testing.T) {
	t.Skip("engine gap: skip combat phase effects not implemented (CR 505.1a)")
}

// CR 506.4a — "A player may activate abilities and cast instant spells during
// the declare attackers step. If a creature that 'can't attack' is attacking,
// it is removed from combat."
func TestRuntimeCantAttackEffects(t *testing.T) {
	t.Skip("engine gap: instant-speed 'can't attack' effects during declare attackers (CR 506.4a)")
}

// CR 506.5 — "'Attacks alone' triggers check if the creature is the only
// attacker declared. 'Blocks alone' triggers check if the creature is the
// only blocker declared."
func TestAttacksAloneSelector(t *testing.T) {
	t.Skip("engine gap: 'attacks alone' / 'blocks alone' selectors not implemented (CR 506.5)")
}

// CR 508.2a — "Blocking restrictions that use colors can be overridden by
// instants or abilities that change creature colors during declare blockers."
func TestColorOverrideBlockingRestrictions(t *testing.T) {
	t.Skip("engine gap: color override for blocking restrictions (CR 508.2a)")
}

// CR 510.1c — "A blocked creature assigns its combat damage to the creatures
// blocking it. If no creatures are currently blocking it, it assigns no combat
// damage. If exactly one creature is blocking it, it assigns all its combat
// damage to that creature. If two or more creatures are blocking it, it assigns
// its combat damage to those creatures according to the damage assignment order."
func TestMultiBlockerDamageAssignment(t *testing.T) {
	t.Skip("engine gap: multi-blocker damage assignment DSL not implemented (CR 510.1c)")
}

// CR 514.3 — "If a state-based action or a triggered ability happens during
// cleanup, players get priority afterward. Then another cleanup step begins."
func TestCleanupPriorityDenial(t *testing.T) {
	t.Skip("engine gap: priority denial in cleanup not fully observable (CR 514.3)")
}

// CR 514.3a — "A cleanup step event should fire so triggers can observe it."
func TestCleanupEvent(t *testing.T) {
	t.Skip("engine gap: EvtCleanup event not implemented (CR 514.3a)")
}
