// Package gametest: combat-restriction tests covering CR 509.1b (block
// restrictions) and CR 509.1c (must-be-blocked-if-able). Exercises the
// engine primitives in pkg/mage/combat_restrictions.go.
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// registerOnce registers a card factory only on first use. Tests run in a
// single process so the global registry is shared; the helper guards against
// duplicate-registration panics across multiple t.Run cases.
func registerOnce(name string, factory func() mage.Card) {
	if !mage.CardRegistered(name) {
		mage.Register(name, factory)
	}
}

// TestCantBeBlockedExceptByFilter exercises Source's "can't be blocked
// except by creatures with X" restriction (Gingerbrute model). Uses Haste
// as the gating keyword for symmetry with the real card.
func TestCantBeBlockedExceptByFilter(t *testing.T) {
	atkName := "ComboRestr Gingerbread Attacker"
	defaultBlk := "ComboRestr Vanilla Blocker"
	hasteBlk := "ComboRestr Haste Blocker"
	registerOnce(atkName, func() mage.Card {
		return mage.NewCreature(atkName, "{1}", 2, 2,
			mage.WithStaticAbility(
				mage.SourceCantBeBlockedExceptBy(mage.HasKeywordFilter(core.Haste)),
			),
		)
	})
	registerOnce(defaultBlk, func() mage.Card {
		return mage.NewCreature(defaultBlk, "{1}", 2, 2)
	})
	registerOnce(hasteBlk, func() mage.Card {
		return mage.NewCreature(hasteBlk, "{1}", 2, 2, mage.WithKeyword(core.Haste))
	})

	t.Run("non-haste blocker can't block", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, defaultBlk)
		tg.Attack(1, PlayerA, atkName)
		tg.Block(1, PlayerB, defaultBlk, atkName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Block was disallowed → 2 damage through.
		tg.AssertLife(PlayerB, 18)
	})

	t.Run("haste blocker can block", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, hasteBlk)
		tg.Attack(1, PlayerA, atkName)
		tg.Block(1, PlayerB, hasteBlk, atkName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 20)
	})
}

// TestCanBlockOnlyFilter exercises a blocker that can only block attackers
// matching a filter (Rishadan Airship model: "can block only creatures with
// flying"). A flying-only blocker shouldn't be able to block a ground
// creature, but should be able to block flyers.
func TestCanBlockOnlyFilter(t *testing.T) {
	flyOnlyBlk := "ComboRestr FlyOnly Airship"
	groundAtk := "ComboRestr Ground Atk"
	flyAtk := "ComboRestr Flying Atk"
	registerOnce(flyOnlyBlk, func() mage.Card {
		return mage.NewCreature(flyOnlyBlk, "{2}", 2, 4,
			mage.WithKeyword(core.Flying),
			mage.WithStaticAbility(
				mage.SourceCanBlockOnly(mage.HasKeywordFilter(core.Flying)),
			),
		)
	})
	registerOnce(groundAtk, func() mage.Card {
		return mage.NewCreature(groundAtk, "{2}", 3, 3)
	})
	registerOnce(flyAtk, func() mage.Card {
		return mage.NewCreature(flyAtk, "{2}", 3, 3, mage.WithKeyword(core.Flying))
	})

	t.Run("can't block ground attacker", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, groundAtk)
		tg.AddCard(core.ZoneBattlefield, PlayerB, flyOnlyBlk)
		tg.Attack(1, PlayerA, groundAtk)
		tg.Block(1, PlayerB, flyOnlyBlk, groundAtk)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 17) // 3 damage through
	})

	t.Run("can block flying attacker", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, flyAtk)
		tg.AddCard(core.ZoneBattlefield, PlayerB, flyOnlyBlk)
		tg.Attack(1, PlayerA, flyAtk)
		tg.Block(1, PlayerB, flyOnlyBlk, flyAtk)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 20)
	})
}

// TestCantBeBlockedByFewerThan exercises Goblin Goon's "can't be blocked
// except by three or more creatures." A single blocker isn't enough.
func TestCantBeBlockedByFewerThan(t *testing.T) {
	goon := "ComboRestr Goon"
	chump1 := "ComboRestr Chump1"
	chump2 := "ComboRestr Chump2"
	chump3 := "ComboRestr Chump3"
	registerOnce(goon, func() mage.Card {
		return mage.NewCreature(goon, "{2}{R}{R}", 6, 6,
			mage.WithStaticAbility(
				mage.SourceCantBeBlockedByFewerThan(3),
			),
		)
	})
	for _, n := range []string{chump1, chump2, chump3} {
		name := n
		registerOnce(name, func() mage.Card {
			return mage.NewCreature(name, "{1}", 1, 1)
		})
	}

	t.Run("two blockers insufficient", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump1)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump2)
		tg.Attack(1, PlayerA, goon)
		tg.Block(1, PlayerB, chump1, goon)
		tg.Block(1, PlayerB, chump2, goon)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Both chumps were dropped from the block (insufficient blockers).
		// 6 damage through.
		tg.AssertLife(PlayerB, 14)
		// Chumps survive — they never actually blocked.
		tg.AssertPermanentCount(PlayerB, chump1, 1)
		tg.AssertPermanentCount(PlayerB, chump2, 1)
	})

	t.Run("three blockers sufficient", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump1)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump2)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump3)
		tg.Attack(1, PlayerA, goon)
		tg.Block(1, PlayerB, chump1, goon)
		tg.Block(1, PlayerB, chump2, goon)
		tg.Block(1, PlayerB, chump3, goon)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Goon ate all three.
		tg.AssertLife(PlayerB, 20)
		tg.AssertPermanentCount(PlayerB, chump1, 0)
		tg.AssertPermanentCount(PlayerB, chump2, 0)
		tg.AssertPermanentCount(PlayerB, chump3, 0)
	})
}

// TestPowerLessOrEqualFilter exercises Ghirapur Guide model:
// "Target creature can't be blocked by creatures with power N or less."
// We attach via TargetCantBeBlockedExceptBy + PowerGreaterThan(N).
func TestPowerLessOrEqualFilter(t *testing.T) {
	atkName := "ComboRestr Guided"
	smallBlk := "ComboRestr SmallBlk"
	largeBlk := "ComboRestr LargeBlk"
	registerOnce(atkName, func() mage.Card {
		return mage.NewCreature(atkName, "{2}", 3, 3)
	})
	registerOnce(smallBlk, func() mage.Card {
		return mage.NewCreature(smallBlk, "{1}", 2, 2)
	})
	registerOnce(largeBlk, func() mage.Card {
		return mage.NewCreature(largeBlk, "{4}", 5, 5)
	})

	t.Run("power-2 blocker can't block when threshold is 2", func(t *testing.T) {
		tg := NewTestGame(t)
		atkID := tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, smallBlk)

		// Install the restriction directly via the engine API.
		eff := mage.TargetCantBeBlockedExceptBy(atkID, mage.PowerGreaterThan(2), core.EndOfTurn)
		eff.SetSourceID(atkID)
		tg.Game.AddContinuousEffect(eff)
		tg.Game.ApplyContinuousEffects()

		tg.Attack(1, PlayerA, atkName)
		tg.Block(1, PlayerB, smallBlk, atkName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 17) // 3 damage through
	})

	t.Run("power-5 blocker can block with same threshold", func(t *testing.T) {
		tg := NewTestGame(t)
		atkID := tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, largeBlk)

		eff := mage.TargetCantBeBlockedExceptBy(atkID, mage.PowerGreaterThan(2), core.EndOfTurn)
		eff.SetSourceID(atkID)
		tg.Game.AddContinuousEffect(eff)
		tg.Game.ApplyContinuousEffects()

		tg.Attack(1, PlayerA, atkName)
		tg.Block(1, PlayerB, largeBlk, atkName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 20)
	})
}

// TestMustBeBlockedIfAble exercises CR 509.1c: a creature with
// AttrMustBeBlockedIfAble must be blocked if the defender controls at least
// one able blocker. Modeled here via TargetMustBeBlockedIfAble (Enlarge,
// Irresistible Prey).
func TestMustBeBlockedIfAble(t *testing.T) {
	atkName := "ComboRestr LureLite Atk"
	blkName := "ComboRestr LureLite Blk"
	registerOnce(atkName, func() mage.Card {
		return mage.NewCreature(atkName, "{2}", 4, 4)
	})
	registerOnce(blkName, func() mage.Card {
		return mage.NewCreature(blkName, "{1}", 2, 2)
	})

	t.Run("forces a block when defender stays passive", func(t *testing.T) {
		tg := NewTestGame(t)
		atkID := tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, blkName)

		eff := mage.TargetMustBeBlockedIfAble(atkID, core.EndOfTurn)
		eff.SetSourceID(atkID)
		tg.Game.AddContinuousEffect(eff)
		tg.Game.ApplyContinuousEffects()

		tg.Attack(1, PlayerA, atkName)
		// Player B doesn't script a block — engine must force one.
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Blocker died (4 vs 2), no damage through to player.
		tg.AssertLife(PlayerB, 20)
		tg.AssertPermanentCount(PlayerB, blkName, 0)
	})

	t.Run("no force when defender has no able blockers", func(t *testing.T) {
		tg := NewTestGame(t)
		atkID := tg.AddCard(core.ZoneBattlefield, PlayerA, atkName)
		// No blockers in play.

		eff := mage.TargetMustBeBlockedIfAble(atkID, core.EndOfTurn)
		eff.SetSourceID(atkID)
		tg.Game.AddContinuousEffect(eff)
		tg.Game.ApplyContinuousEffects()

		tg.Attack(1, PlayerA, atkName)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 16) // unblocked, 4 damage through
	})
}

// TestPreventBlockByPowerLessThanSource exercises Champion of Lambholt:
// "Creatures with power less than ~'s power can't block creatures you
// control."
func TestPreventBlockByPowerLessThanSource(t *testing.T) {
	champ := "ComboRestr Lambholt-like"
	teamMate := "ComboRestr TeamMate"
	smallBlk := "ComboRestr Small"
	largeBlk := "ComboRestr Large"
	registerOnce(champ, func() mage.Card {
		return mage.NewCreature(champ, "{2}{G}", 3, 3,
			mage.WithStaticAbility(
				mage.PreventBlockByPowerLessThanSource(mage.AnyPermanent),
			),
		)
	})
	registerOnce(teamMate, func() mage.Card {
		return mage.NewCreature(teamMate, "{2}", 2, 2)
	})
	registerOnce(smallBlk, func() mage.Card {
		return mage.NewCreature(smallBlk, "{1}", 2, 2)
	})
	registerOnce(largeBlk, func() mage.Card {
		return mage.NewCreature(largeBlk, "{3}", 4, 4)
	})

	t.Run("smaller-power blocker can't block teammate", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, champ)
		tg.AddCard(core.ZoneBattlefield, PlayerA, teamMate)
		tg.AddCard(core.ZoneBattlefield, PlayerB, smallBlk)
		tg.Attack(1, PlayerA, teamMate)
		tg.Block(1, PlayerB, smallBlk, teamMate)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Block disallowed (smallBlk power 2 < champ power 3) → 2 damage.
		tg.AssertLife(PlayerB, 18)
	})

	t.Run("equal-or-greater blocker can block", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, champ)
		tg.AddCard(core.ZoneBattlefield, PlayerA, teamMate)
		tg.AddCard(core.ZoneBattlefield, PlayerB, largeBlk)
		tg.Attack(1, PlayerA, teamMate)
		tg.Block(1, PlayerB, largeBlk, teamMate)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 20)
	})
}

// TestPreventAttackingIfDefenderControlsMore exercises the Goblin Goon
// attack-restriction primitive: source can't attack if the defender
// controls strictly more permanents matching filter than the source's
// controller does.
func TestPreventAttackingIfDefenderControlsMore(t *testing.T) {
	goon := "ComboRestr Goon-Atk"
	registerOnce(goon, func() mage.Card {
		return mage.NewCreature(goon, "{2}{R}{R}", 6, 6,
			mage.WithStaticAbility(
				mage.PreventAttackingIfDefenderControlsMore(mage.IsCreature),
			),
		)
	})
	chump := "ComboRestr Goon-Chump"
	registerOnce(chump, func() mage.Card {
		return mage.NewCreature(chump, "{1}", 1, 1)
	})

	t.Run("can't attack when outnumbered", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump, 2)
		tg.Attack(1, PlayerA, goon)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Attack was suppressed.
		tg.AssertLife(PlayerB, 20)
	})

	t.Run("can attack when not outnumbered", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerA, chump)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump)
		tg.Attack(1, PlayerA, goon)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 14)
	})
}

// TestPreventAttackingIfDefenderControlsAsManyOrMore exercises the
// strict "more creatures than defending player" variant: equality also
// revokes the attack.
func TestPreventAttackingIfDefenderControlsAsManyOrMore(t *testing.T) {
	goon := "ComboRestr Goon-AtkStrict"
	registerOnce(goon, func() mage.Card {
		return mage.NewCreature(goon, "{2}{R}{R}", 6, 6,
			mage.WithStaticAbility(
				mage.PreventAttackingIfDefenderControlsAsManyOrMore(mage.IsCreature),
			),
		)
	})
	chump := "ComboRestr Goon-ChumpStrict"
	registerOnce(chump, func() mage.Card { return mage.NewCreature(chump, "{1}", 1, 1) })

	t.Run("can't attack when equal", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump)
		tg.Attack(1, PlayerA, goon)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 20)
	})

	t.Run("can attack when strictly more", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerA, chump)
		tg.AddCard(core.ZoneBattlefield, PlayerB, chump)
		tg.Attack(1, PlayerA, goon)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertLife(PlayerB, 14)
	})
}

// TestPreventBlockingIfAttackerControlsAsManyOrMore exercises the
// block-side restriction: source can't be declared a blocker unless
// the source's controller controls strictly more permanents matching
// the filter than the attacking player does.
func TestPreventBlockingIfAttackerControlsAsManyOrMore(t *testing.T) {
	goon := "ComboRestr Goon-Blk"
	registerOnce(goon, func() mage.Card {
		return mage.NewCreature(goon, "{2}{R}{R}", 6, 6,
			mage.WithStaticAbility(
				mage.PreventBlockingIfAttackerControlsAsManyOrMore(mage.IsCreature),
			),
		)
	})
	bear := "ComboRestr Goon-Blk-Bear"
	registerOnce(bear, func() mage.Card { return mage.NewCreature(bear, "{1}{G}", 2, 2) })

	t.Run("can't block when equal", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerB, goon)
		tg.AddCard(core.ZoneBattlefield, PlayerA, bear)
		tg.Attack(1, PlayerA, bear)
		tg.Block(1, PlayerB, goon, bear)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// Block was suppressed; bear got through.
		tg.AssertLife(PlayerB, 18)
	})
}
