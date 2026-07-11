package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// TestOrmos_EmptyLibraryDrawBecomesCounters verifies the replacement
// effect: when PlayerA's library has been drained and the turn-based
// draw would fire, the draw is replaced with five +1/+1 counters on
// Ormos and PlayerA does not lose to drawing from an empty library.
func TestOrmos_EmptyLibraryDrawBecomesCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ormos, Archive Keeper")
	// Seed PlayerA's library with a single non-padding card so
	// padLibraries() (which only fires on empty libraries) leaves the
	// library exactly one card deep. PlayerA draws this card on turn 3
	// and then has an empty library for turn 5's draw.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains", 60)
	g.StopAt(5, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Ormos, Archive Keeper", core.P1P1, 5)
	// PlayerA must still be in the game (replacement prevented the
	// drewFromEmpty SBA loss on turn 5).
	g.AssertGameOver(false)
}

// TestOrmos_NonEmptyLibraryNormalDraw verifies that with cards in
// library, the draw step proceeds normally and Ormos receives no
// counters.
func TestOrmos_NonEmptyLibraryNormalDraw(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ormos, Archive Keeper")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant", 10)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains", 60)
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Ormos, Archive Keeper", core.P1P1, 0)
	g.AssertPermanentCount(gametest.PlayerA, "Ormos, Archive Keeper", 1)
}

// TestOrmos_ActivatedAbilityDrawsFive verifies the activated ability's
// discard-three-different-names cost and "Draw five cards" effect.
func TestOrmos_ActivatedAbilityDrawsFive(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ormos, Archive Keeper")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	// Use a non-padding-named card in the library so we can count the
	// drawn cards distinctly from anything padLibraries might add.
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 10)
	g.ChooseDiscard(gametest.PlayerA, "Mountain", "Lightning Bolt", "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ormos, Archive Keeper")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	// Activated ability: Discard 3 (cost) → Draw 5 (effect). 5 Forests
	// move out of library; 5 remain. Note PlayerA may auto-play one
	// Forest as a land during the postcombat main, so check total
	// Forests across hand + battlefield instead of asserting hand only.
	pA := g.GetPlayer(gametest.PlayerA)
	g.AssertLibraryCount(gametest.PlayerA, "Forest", 5)
	forestsHand := 0
	for _, c := range pA.Hand() {
		if c.Name() == "Forest" {
			forestsHand++
		}
	}
	forestsBF := 0
	for _, perm := range g.AllBattlefield() {
		if perm.Card.Name() == "Forest" && perm.ControllerID() == pA.PlayerID() {
			forestsBF++
		}
	}
	if got := forestsHand + forestsBF; got != 5 {
		t.Errorf("expected 5 Forests in hand+battlefield, got %d (hand=%d bf=%d)", got, forestsHand, forestsBF)
	}
}

// TestOrmos_ActivatedAbilityBlockedWhenInsufficientDistinctNames
// verifies that with only two distinct names in hand, the activated
// ability cannot be paid and no card is discarded as a cost.
func TestOrmos_ActivatedAbilityBlockedWhenInsufficientDistinctNames(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ormos, Archive Keeper")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest", 10)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ormos, Archive Keeper")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// The cost can't be paid (only 2 distinct names), so the cards
	// should not have been discarded as a cost. Hill Giant is a
	// creature and Lightning Bolt is an instant, neither can be
	// auto-played, so they remain in hand.
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 2)
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
}
