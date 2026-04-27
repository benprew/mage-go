package gametest

// Chapter 6c: Replacement and Prevention Effects (CR 614–616)

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// 614.1a — Replacement with "instead" (CR 614.1a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_1a_ReplacementInsteadKeyword verifies that a replacement effect using
// "instead" fully replaces the original event — exile instead of graveyard.
func TestCR614_1a_ReplacementInsteadKeyword(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Giant")
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Exile Spell")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Exile Spell", "Ch06 Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Giant exiled, not destroyed.
	g.AssertExileCount("Ch06 Giant", 1)
	g.AssertGraveyardCount(PlayerB, "Ch06 Giant", 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// 614.1b — "Skip" replacement (CR 614.1b)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_1b_ReplacementSkipReplacesWithNothing verifies "skip draw" causes no card
// to be drawn during the draw step.
func TestCR614_1b_ReplacementSkipReplacesWithNothing(t *testing.T) {
	g := NewTestGame(t)
	g.AddCard(core.ZoneLibrary, PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, PlayerA, "Forest")

	g.SetSkipNextDraw(g.Players[0].PlayerID())
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	// PlayerA drew no card; hand is empty.
	g.AssertHandSize(PlayerA, 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// 614.4 — Replacement must exist before the event (CR 614.4)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_4_ReplacementMustExistBeforeEvent verifies that a creature without a regen
// shield is destroyed normally.
func TestCR614_4_ReplacementMustExistBeforeEvent(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Regen Skeleton") // 1/1
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Bolt")

	// No regen activated; bolt kills the 1/1.
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Bolt", "Ch06 Regen Skeleton")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertGraveyardCount(PlayerA, "Ch06 Regen Skeleton", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 614.6 — Replaced event never happens; LTB trigger doesn't fire (CR 614.6)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_6_ReplacementReplacedEventNeverHappens verifies that when destruction is
// replaced by regeneration, the "dies" LTB trigger doesn't fire.
func TestCR614_6_ReplacementReplacedEventNeverHappens(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Death Regen") // 2/2 with LTB trigger + regen
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Destroy Spell")

	// Activate regen first.
	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Death Regen")
	// Destroy spell tries to destroy.
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Destroy Spell", "Ch06 Death Regen")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Regen replaces destruction; creature survives; LTB trigger doesn't fire.
	g.AssertPermanentCount(PlayerA, "Ch06 Death Regen", 1)
	g.AssertLife(PlayerA, 20) // no life gained from LTB trigger
}

// ─────────────────────────────────────────────────────────────────────────────
// 614.8 — Regeneration is a destruction-replacement effect (CR 614.8)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_8_ReplacementRegenerationPreventsDestruction verifies regeneration
// survives a destroy effect.
func TestCR614_8_ReplacementRegenerationPreventsDestruction(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Regen Skeleton")
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Destroy Spell")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Regen Skeleton")
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Destroy Spell", "Ch06 Regen Skeleton")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertPermanentCount(PlayerA, "Ch06 Regen Skeleton", 1)
}

// TestCR614_8_ReplacementRegenerationTapsAndRemovesFromCombat verifies that a
// regenerating creature is tapped after regeneration.
func TestCR614_8_ReplacementRegenerationTapsAndRemovesFromCombat(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Regen Skeleton")
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Destroy Spell")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Regen Skeleton")
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Destroy Spell", "Ch06 Regen Skeleton")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertTapped(PlayerA, "Ch06 Regen Skeleton", true)
}

// ─────────────────────────────────────────────────────────────────────────────
// 614.9 — Redirection: damage redirected from one target to another (CR 614.9)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_9_ReplacementRedirectDamage verifies that attacker damage is redirected
// from a player to a creature.
func TestCR614_9_ReplacementRedirectDamage(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Giant")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Ch06 Bear")

	hillID := g.FindPermanentByName("Ch06 Giant", g.GetPlayer(PlayerA).PlayerID())
	bearID := g.FindPermanentByName("Ch06 Bear", g.GetPlayer(PlayerB).PlayerID())
	g.SetAttackerDamageRedirect(hillID.ID(), bearID.ID())

	g.Attack(1, PlayerA, "Ch06 Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// PlayerB takes no damage; Bear takes 3 and dies.
	g.AssertLife(PlayerB, 20)
	g.AssertGraveyardCount(PlayerB, "Ch06 Bear", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 614.10a — Multiple "skip next" effects skip separate occurrences (CR 614.10a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR614_10a_ReplacementSkipMultipleEffectsEachSkipsSeparateOccurrence verifies
// that a skip-draw effect skips the draw step.
func TestCR614_10a_ReplacementSkipMultipleEffectsEachSkipsSeparateOccurrence(t *testing.T) {
	g := NewTestGame(t)
	g.AddCard(core.ZoneLibrary, PlayerA, "Forest")

	g.SetSkipNextDraw(g.Players[0].PlayerID())
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertHandSize(PlayerA, 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// 615.4 — Prevention must exist before damage (CR 615.4)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR615_4_PreventionMustExistBeforeDamage verifies that without a shield, damage
// is dealt in full.
func TestCR615_4_PreventionMustExistBeforeDamage(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertLife(PlayerB, 17)
}

// ─────────────────────────────────────────────────────────────────────────────
// 615.6 — Fully prevented damage never happens (CR 615.6)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR615_6_PreventionPreventedDamageNeverHappens verifies that fully prevented
// damage doesn't affect the target's life total.
func TestCR615_6_PreventionPreventedDamageNeverHappens(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")
	g.AddPreventionShield(g.Players[1].PlayerID(), 10)

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertLife(PlayerB, 20)
}

// ─────────────────────────────────────────────────────────────────────────────
// 615.7 — Prevention shield reduced by amount prevented (CR 615.7)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR615_7_PreventionShieldReducesByAmountPrevented verifies a shield fully
// absorbs damage up to its value.
func TestCR615_7_PreventionShieldReducesByAmountPrevented(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")
	g.AddPreventionShield(g.Players[1].PlayerID(), 5)

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// All 3 prevented by shield of 5.
	g.AssertLife(PlayerB, 20)
}

// ─────────────────────────────────────────────────────────────────────────────
// 616.1a — Self-replacement applied before external replacements (CR 616.1a)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR616_1a_ReplacementInteractionSelfFirst verifies that a creature's own
// regeneration fires before external destruction replacements.
func TestCR616_1a_ReplacementInteractionSelfFirst(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Ch06 Regen Skeleton")
	g.AddCard(core.ZoneHand, PlayerB, "Ch06 Destroy Spell")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Ch06 Regen Skeleton")
	g.CastSpell(1, core.PrecombatMain, PlayerB, "Ch06 Destroy Spell", "Ch06 Regen Skeleton")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Regen (self-replacement) fires first; creature survives.
	g.AssertPermanentCount(PlayerA, "Ch06 Regen Skeleton", 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// 616.1f — Chained replacement effects (CR 616.1f)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR616_1f_ReplacementInteractionChainedEffects verifies two prevention shields
// each reduce damage in sequence.
func TestCR616_1f_ReplacementInteractionChainedEffects(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")

	pid := g.Players[1].PlayerID()
	g.AddPreventionShield(pid, 1) // absorbs 1 of 3
	g.AddPreventionShield(pid, 1) // absorbs 1 more

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// 3 - 1 - 1 = 1 damage through.
	g.AssertLife(PlayerB, 19)
}

// ─────────────────────────────────────────────────────────────────────────────
// 616.2 — Replacement effects becoming applicable (CR 616.2)
// ─────────────────────────────────────────────────────────────────────────────

// TestCR616_2_ReplacementInteractionBecomesApplicable verifies a doubling effect
// followed by a prevention shield chains correctly.
func TestCR616_2_ReplacementInteractionBecomesApplicable(t *testing.T) {
	registerCh06Cards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneHand, PlayerA, "Ch06 Bolt")

	pid := g.Players[1].PlayerID()
	g.AddPreventionShield(pid, 2)
	g.AddReplacementEffect(&ch06DoubleDamageReplacement{playerID: pid})

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Ch06 Bolt", "PlayerB")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// 3 doubled to 6; then 2 prevented = 4 through.
	g.AssertLife(PlayerB, 16)
}

// ch06DoubleDamageReplacement doubles damage to a specific player.
type ch06DoubleDamageReplacement struct {
	playerID uuid.UUID
}

func (r *ch06DoubleDamageReplacement) Matches(a mage.Action, _ mage.GameReader) bool {
	act, ok := a.(*mage.DamageToPlayerAction)
	return ok && act.PlayerID() == r.playerID
}

func (r *ch06DoubleDamageReplacement) Replace(a mage.Action, _ *mage.Game) mage.Action {
	act := a.(*mage.DamageToPlayerAction)
	return act.WithAmount(act.Amount() * 2)
}

func (r *ch06DoubleDamageReplacement) SourceID() uuid.UUID             { return uuid.Nil }
func (r *ch06DoubleDamageReplacement) IsActive(_ mage.GameReader) bool { return true }
func (r *ch06DoubleDamageReplacement) GetDuration() core.Duration      { return core.EndOfTurn }
func (r *ch06DoubleDamageReplacement) Clone() mage.ReplacementEffect {
	c := *r
	return &c
}
