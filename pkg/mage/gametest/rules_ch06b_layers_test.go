package gametest

// Chapter 6b: Layer System (CR 613)

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// 613 — Layer 2: Control changes (CR 613)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_Layer2ControlChangeTimestamp verifies that a control-change aura
// gives control of the creature to its controller.
func TestCR613_Layer2ControlChangeTimestamp(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Control Aura")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Control Aura", "Ch06 Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// PlayerA now controls Ch06 Giant.
	g.AssertPermanentCount(PlayerA, "Ch06 Giant", 1)
	g.AssertPermanentCount(PlayerB, "Ch06 Giant", 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// 613 — Layer 4: Type-changing effects (CR 613)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_Layer4TypeChangeTurnLandIntoCreature verifies that a type-change
// effect animates a Forest into a 1/1 creature.
func TestCR613_Layer4TypeChangeTurnLandIntoCreature(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Living Lands")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertPowerToughness(PlayerA, "Forest", 1, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 613 — Layer 6: Ability adding (CR 613)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_Layer6AbilityAddTimestamp verifies that an aura granting flying
// applies in layer 6 so the creature has flying.
func TestCR613_Layer6AbilityAddTimestamp(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")
	flightID := g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Flight")

	g.Attach(flightID, bearID)
	g.Effects.Apply(g.Game)

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertHasAbility(PlayerA, "Ch06 Bear", core.Flying, true)
}

// ─────────────────────────────────────────────────────────────────────────────
// 613 — Layer 7b set P/T, then 7c modify (CR 613)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_Layer7bSetPTThenModify7c verifies that setting base P/T in layer 7b
// and then applying a +P/+T boost in 7c gives the correct final value.
func TestCR613_Layer7bSetPTThenModify7c(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Sorceress Queen")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear") // 2/2
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Growth")

	// Set Bear to 0/2 (layer 7b), then +3/+3 (layer 7c) = 3/5.
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Sorceress Queen", "Ch06 Bear")
	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Growth", "Ch06 Bear")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPowerToughness(PlayerA, "Ch06 Bear", 3, 5)
}

// ─────────────────────────────────────────────────────────────────────────────
// 613.5 — Layer system applied continuously and instantaneously (CR 613.5)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_ContinuousApplicationDynamic verifies that when a creature
// becomes white, it immediately gains the bonus from Crusade.
func TestCR613_ContinuousApplicationDynamic(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Crusade")
	bearID := g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear") // green 2/2
	paintID := g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 White Paint")

	g.Attach(paintID, bearID)
	g.Effects.Apply(g.Game)

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Bear became white via aura; Crusade applies: 3/3.
	g.AssertPowerToughness(PlayerA, "Ch06 Bear", 3, 3)
}

// ─────────────────────────────────────────────────────────────────────────────
// 613.6 — Multi-layer effect: type in layer 4 and P/T in 7b (CR 613.6)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_MultiLayerTypeAndPT verifies that Kormus Bell applies type change
// (layer 4) and P/T (layer 7b) to the same permanents.
func TestCR613_MultiLayerTypeAndPT(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Kormus Bell")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Swamp")

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Swamp becomes a 1/1 creature.
	g.AssertPowerToughness(PlayerA, "Swamp", 1, 1)
}

// TestCR613_Layer2ControlChangeCanAttack verifies that a stolen creature can
// attack for its new controller on subsequent turns.
func TestCR613_Layer2ControlChangeCanAttack(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Control Aura")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Control Aura", "Ch06 Giant")
	g.Attack(3, PlayerA, "Ch06 Giant") // turn 3 is PlayerA's turn
	g.StopAt(3, core.EndCombat)
	g.Execute()

	// Giant attacks for PlayerA; PlayerB takes 3 damage.
	g.AssertLife(PlayerB, 17)
}

// ─────────────────────────────────────────────────────────────────────────────
// 613.7d — Object receives timestamp when it enters a zone (CR 613.7d)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR613_7d_TimestampEntersBattlefield verifies that a later-entering permanent's
// static effect applies normally alongside an earlier one.
func TestCR613_7d_TimestampEntersBattlefield(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bad Moon")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Black Knight") // 2/2 black

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// Bad Moon boosts Black Knight to 3/3.
	g.AssertPowerToughness(PlayerA, "Ch06 Black Knight", 3, 3)
}

// ─────────────────────────────────────────────────────────────────────────────
