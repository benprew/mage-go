package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var perTurnTrackerOnce sync.Once

func registerPerTurnTrackerCards() {
	perTurnTrackerOnce.Do(func() {
		// A sorcery whose additional cost is "discard a card" — used to drive
		// PlayerDiscardCountThisTurn for the caster.
		if !mage.CardRegistered("Test Discard Self") {
			mage.Register("Test Discard Self", func() mage.Card {
				return mage.NewSorcery("Test Discard Self", "{B}",
					mage.NewSpellAbility(mage.GainLife(1)),
					mage.WithAdditionalCost(mage.DiscardCost(1)),
				)
			})
		}
		// A sorcery whose controller gains 3 life.
		if !mage.CardRegistered("Test Gain 3 Life") {
			mage.Register("Test Gain 3 Life", func() mage.Card {
				return mage.NewSorcery("Test Gain 3 Life", "{W}",
					mage.NewSpellAbility(mage.GainLife(3)),
				)
			})
		}
		// A bolt-like sorcery for damage tests.
		if !mage.CardRegistered("Test Bolt 3") {
			mage.Register("Test Bolt 3", func() mage.Card {
				return mage.NewSorcery("Test Bolt 3", "{R}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(3))),
				)
			})
		}
		if !mage.CardRegistered("Test Tracker Bear") {
			mage.Register("Test Tracker Bear", func() mage.Card {
				return mage.NewCreature("Test Tracker Bear", "{1}{G}", 4, 4,
					mage.WithSubTypes("Bear"))
			})
		}
	})
}

// TestPlayerLifeGainedThisTurn verifies the per-turn life-gained accessor
// counts EvtLifeGained amounts.
func TestPlayerLifeGainedThisTurn(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Gain 3 Life")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Gain 3 Life")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Gain 3 Life")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Gain 3 Life")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	pid := tg.GetPlayer(PlayerA).PlayerID()
	if got := tg.Game.PlayerLifeGainedThisTurn(pid); got != 6 {
		t.Errorf("PlayerLifeGainedThisTurn = %d, want 6", got)
	}
}

// TestPlayerDiscardCountThisTurn verifies the per-turn discard accessor.
func TestPlayerDiscardCountThisTurn(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Plains")
	tg.AddCard(core.ZoneHand, PlayerA, "Plains")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Discard Self")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Discard Self")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	pid := tg.GetPlayer(PlayerA).PlayerID()
	if got := tg.Game.PlayerDiscardCountThisTurn(pid); got != 1 {
		t.Errorf("PlayerDiscardCountThisTurn = %d, want 1", got)
	}
}

// TestPermanentDamageReceivedThisTurn verifies the per-permanent damage
// tracker.
func TestPermanentDamageReceivedThisTurn(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	bearID := tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Tracker Bear")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Bolt 3")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Bolt 3", "Test Tracker Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	if got := tg.Game.PermanentDamageReceivedThisTurn(bearID); got != 3 {
		t.Errorf("PermanentDamageReceivedThisTurn = %d, want 3", got)
	}
}

// TestPerTurnTrackersResetEachTurn confirms cleanup wipes the trackers.
func TestPerTurnTrackersResetEachTurn(t *testing.T) {
	registerPerTurnTrackerCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Gain 3 Life")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Gain 3 Life")
	// Run two full turns; on turn 2 we don't gain life again.
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	pid := tg.GetPlayer(PlayerA).PlayerID()
	if got := tg.Game.PlayerLifeGainedThisTurn(pid); got != 0 {
		t.Errorf("life-gained should reset between turns; got %d", got)
	}
}
