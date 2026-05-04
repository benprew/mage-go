package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestGrantExtraLandPlay_AllowsSecondLand verifies that GrantExtraLandPlay
// raises the active player's land-play allowance for the current turn so
// they can legally play a second land. Without the grant, the second
// PlayLand would be rejected by the CR 505.6b limit.
func TestGrantExtraLandPlay_AllowsSecondLand(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.AddCard(core.ZoneHand, PlayerA, "Forest", 2)

	tg.Turn = 1
	tg.ActivePlayer = 0
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	for _, step := range []core.PhaseStep{core.Untap, core.Upkeep, core.Draw} {
		tg.Step = step
		tg.RunStepWithPriority(step)
	}
	tg.Step = core.PrecombatMain

	playerID := tg.getPlayerID(PlayerA)
	tg.GrantExtraLandPlay(playerID, 1)

	if got := tg.MaxLandPlays(); got != 2 {
		t.Errorf("MaxLandPlays after GrantExtraLandPlay(1): got %d want 2", got)
	}

	first, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: first Forest not in hand")
	}
	if err := tg.PlayLand(playerID, first.ID()); err != nil {
		t.Fatalf("first land play failed: %v", err)
	}

	second, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: second Forest not in hand")
	}
	if err := tg.PlayLand(playerID, second.ID()); err != nil {
		t.Fatalf("second land play (with extra grant) failed: %v", err)
	}
	tg.AssertPermanentCount(PlayerA, "Forest", 2)
}

// TestGrantExtraLandPlay_ResetAtEndOfTurn verifies that the per-turn
// allowance is cleared during cleanup so the next turn starts at the
// default 1-land limit.
func TestGrantExtraLandPlay_ResetAtEndOfTurn(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()

	playerID := tg.getPlayerID(PlayerA)
	tg.GrantExtraLandPlay(playerID, 2)
	if got := tg.ExtraLandPlaysGrantedThisTurn(playerID); got != 2 {
		t.Errorf("ExtraLandPlaysGrantedThisTurn: got %d want 2", got)
	}

	tg.StopAt(2, core.Upkeep)
	tg.Execute()

	if got := tg.ExtraLandPlaysGrantedThisTurn(playerID); got != 0 {
		t.Errorf("ExtraLandPlaysGrantedThisTurn after turn flip: got %d want 0", got)
	}
	if got := tg.MaxLandPlays(); got != 1 {
		t.Errorf("MaxLandPlays after reset: got %d want 1", got)
	}
}

// TestGrantExtraLandPlay_Cumulative verifies that multiple grants stack
// (e.g. two Explore copies grant +2 total).
func TestGrantExtraLandPlay_Cumulative(t *testing.T) {
	tg := NewTestGame(t)
	playerID := tg.getPlayerID(PlayerA)
	tg.GrantExtraLandPlay(playerID, 1)
	tg.GrantExtraLandPlay(playerID, 2)

	if got := tg.ExtraLandPlaysGrantedThisTurn(playerID); got != 3 {
		t.Errorf("cumulative grant: got %d want 3", got)
	}
	if got := tg.MaxLandPlays(); got != 4 {
		t.Errorf("MaxLandPlays cumulative: got %d want 4", got)
	}
}
