package gametest

// Chapter 6d: Mana Abilities, Effects, and Continuous Effects (CR 605, 609, 611)

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// 605.1a — Activated mana ability doesn't use the stack (CR 605.1a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR605_1a_ManaAbilityActivatedNoStack verifies tapping a land for mana produces
// mana without using the stack.
func TestCR605_1a_ManaAbilityActivatedNoStack(t *testing.T) {
	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Forest")
	// Stop at BeginCombat so PrecombatMain runs fully and the activation fires.
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertManaProducedAtLeast(PlayerA, core.Green, 1)
	g.AssertTapped(PlayerA, "Forest", true)
}

// TestCR605_3a_ManaAbilityActivatableDuringCasting verifies that mana can be produced
// during casting to pay for the spell (CR 605.3a).
func TestCR605_3a_ManaAbilityActivatableDuringCasting(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Growth")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Growth", "Ch06 Bear")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Ch06 Bear", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 609.3 — Impossible effect does as much as possible (CR 609.3)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR609_3_EffectImpossibleDoesAsMuchAsPossible verifies that "discard 2" with
// only 1 card discards as many as possible.
func TestCR609_3_EffectImpossibleDoesAsMuchAsPossible(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Discard Two")
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Bear") // only 1 card in hand

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Discard Two", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// PlayerB had 1 card; discarded it (did as much as possible).
	g.AssertGraveyardCount(PlayerB, "Ch06 Bear", 1)
}

// 611.2a — "Until end of turn" effect expires (CR 611.2a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR611_2a_ContinuousStatedDuration verifies that Giant Growth's +3/+3 applies
// during the turn it was cast.
func TestCR611_2a_ContinuousStatedDuration(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Growth")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Growth", "Ch06 Bear")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPowerToughness(PlayerA, "Ch06 Bear", 5, 5)
}

// TestCR611_2a_ContinuousEffectExpiresAtEndOfTurn verifies that the effect expires by
// the next turn.
func TestCR611_2a_ContinuousEffectExpiresAtEndOfTurn(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Growth")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Growth", "Ch06 Bear")
	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	// Effect expired; Bear is back to 2/2.
	g.AssertPowerToughness(PlayerA, "Ch06 Bear", 2, 2)
}

// ─────────────────────────────────────────────────────────────────────────────
// 611.2c — Set locked in at resolution for characteristic changes (CR 611.2c)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR611_2c_ContinuousSetLockedInAtResolution verifies that a "boost all white
// creatures until end of turn" sorcery doesn't apply to non-white creatures.
func TestCR611_2c_ContinuousSetLockedInAtResolution(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Knight") // white 2/2
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bear")   // green 2/2
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Rally")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Rally")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// White Knight gets +1/+1 (3/3); Bear (non-white) stays 2/2.
	g.AssertPowerToughness(PlayerA, "Ch06 Knight", 3, 3)
	g.AssertPowerToughness(PlayerA, "Ch06 Bear", 2, 2)
}

// ─────────────────────────────────────────────────────────────────────────────
// 611.3a — Static effect not locked in; applies to future objects (CR 611.3a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR611_3a_ContinuousStaticNotLockedIn verifies a static effect applies to
// creatures matching its condition at any moment.
func TestCR611_3a_ContinuousStaticNotLockedIn(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Bad Moon")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Black Knight") // 2/2 black -> 3/3

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertPowerToughness(PlayerA, "Ch06 Black Knight", 3, 3)
}

// TestCR611_3c_ContinuousStaticAppliesOnEntry verifies that a static effect applies
// simultaneously with a permanent entering (CR 611.3c).
func TestCR611_3c_ContinuousStaticAppliesOnEntry(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Crusade")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Knight")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Knight")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Knight enters; immediately gets +1/+1 from Crusade.
	g.AssertPowerToughness(PlayerA, "Ch06 Knight", 3, 3)
}

// ─────────────────────────────────────────────────────────────────────────────
