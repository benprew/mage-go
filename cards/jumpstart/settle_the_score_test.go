package jumpstart

import (
	"sync"
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Settle the Score {2}{B}{B}
// Sorcery
// Exile target creature. Put two loyalty counters on a planeswalker you control.

var settleTheScoreTestRegistered sync.Once

// registerSettleTheScoreTestCards registers a minimal test-only planeswalker
// used to exercise the planeswalker primitive. Loyalty-activated abilities are
// not implemented; this card is purely a recipient of loyalty counters.
func registerSettleTheScoreTestCards() {
	settleTheScoreTestRegistered.Do(func() {
		if !CardRegistered("Test Walker") {
			Register("Test Walker", func() Card {
				return NewPlaneswalker("Test Walker", "{3}", 3,
					WithSubTypes("Test"),
				)
			})
		}
	})
}

// TestSettleTheScore_ExilesCreatureWithoutPlaneswalker confirms the spell
// resolves and exiles its target even when its controller has no planeswalker
// on the battlefield (the second sentence simply does nothing).
func TestSettleTheScore_ExilesCreatureWithoutPlaneswalker(t *testing.T) {
	registerSettleTheScoreTestCards()

	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Settle the Score")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Settle the Score", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertExileCount("Grizzly Bears", 1)
}

// TestSettleTheScore_AddsLoyaltyToPlaneswalker covers the full Oracle text:
// the creature is exiled and a controlled planeswalker gains two loyalty
// counters. Test Walker enters with 3 loyalty and ends at 5.
func TestSettleTheScore_AddsLoyaltyToPlaneswalker(t *testing.T) {
	registerSettleTheScoreTestCards()

	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Test Walker")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Settle the Score")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Settle the Score", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Grizzly Bears", 1)
	g.AssertCounterCount(gametest.PlayerA, "Test Walker", core.Loyalty, 5)
}

// TestSettleTheScore_PlaneswalkerEntersWithStartingLoyalty exercises the
// EntersWithNCounters(Loyalty, …) replacement embedded in NewPlaneswalker.
func TestSettleTheScore_PlaneswalkerEntersWithStartingLoyalty(t *testing.T) {
	registerSettleTheScoreTestCards()

	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Test Walker")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Test Walker", 1)
	g.AssertCounterCount(gametest.PlayerA, "Test Walker", core.Loyalty, 3)
}

// TestSettleTheScore_PlaneswalkerDiesAtZeroLoyalty exercises CR 704.5i. We
// drop Test Walker's loyalty to zero and verify the next state-based action
// pass moves it to its owner's graveyard.
func TestSettleTheScore_PlaneswalkerDiesAtZeroLoyalty(t *testing.T) {
	registerSettleTheScoreTestCards()

	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Test Walker")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	walker := g.FindPermanentByName("Test Walker", pid)
	if walker == nil {
		t.Fatal("setup: Test Walker not on battlefield")
	}
	walker.Counters[core.Loyalty] = 0
	g.CheckStateBasedActions()

	g.AssertPermanentCount(gametest.PlayerA, "Test Walker", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Test Walker", 1)
}
