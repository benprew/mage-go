// Package gametest: turn-structure tests for the main phases.
// Covers CR 505 (main phases), including 505.1, 505.1a, 505.6a, 505.6b.
package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Register cards used only by this file.
func init() {
	if !mage.CardRegistered("Forest") {
		mage.Register("Forest", func() mage.Card {
			return mage.NewLand("Forest",
				mage.WithSubTypes("Forest"),
				mage.WithManaAbility(core.Green),
			)
		})
	}
	if !mage.CardRegistered("TurnStructureMain Healing Salve Sorcery") {
		mage.Register("TurnStructureMain Healing Salve Sorcery", func() mage.Card {
			return mage.NewSorcery(
				"TurnStructureMain Healing Salve Sorcery",
				"{W}",
				mage.NewSpellAbility(mage.GainLife(3)),
			)
		})
	}
}

// findLandInHand returns the ID of the first land named `name` in the given player's hand.
func findLandInHand(tg *TestGame, p PlayerRef, name string) (mage.Card, bool) {
	for _, c := range tg.GetPlayer(p).Hand() {
		if c.Name() == name {
			return c, true
		}
	}
	return nil, false
}

// TestTurnStructureMain_TwoMainPhasesPerTurn verifies CR 505.1: there are two
// main phases in a turn, precombat and postcombat.
func TestTurnStructureMain_TwoMainPhasesPerTurn(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()

	var visited []core.PhaseStep
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}

	// Drive one full turn manually and record each step visited.
	for _, step := range core.AllSteps() {
		tg.Step = step
		visited = append(visited, step)
		tg.RunStepWithPriority(step)
	}

	// Assert both main phases appear in the turn order, with PrecombatMain
	// before PostcombatMain.
	preIdx, postIdx := -1, -1
	for i, s := range visited {
		if s == core.PrecombatMain && preIdx == -1 {
			preIdx = i
		}
		if s == core.PostcombatMain && postIdx == -1 {
			postIdx = i
		}
	}
	if preIdx == -1 {
		t.Error("CR 505.1: PrecombatMain did not occur in a turn")
	}
	if postIdx == -1 {
		t.Error("CR 505.1: PostcombatMain did not occur in a turn")
	}
	if preIdx >= postIdx {
		t.Errorf("CR 505.1: expected PrecombatMain before PostcombatMain, got indices pre=%d post=%d", preIdx, postIdx)
	}

	// Both main phases must be classified as main phases (sanity for 505.1).
	if !core.PrecombatMain.IsMainPhase() {
		t.Error("CR 505.1: PrecombatMain should be classified as a main phase")
	}
	if !core.PostcombatMain.IsMainPhase() {
		t.Error("CR 505.1: PostcombatMain should be classified as a main phase")
	}
}

// TestTurnStructureMain_PostcombatMainAfterSkippedCombat verifies CR 505.1a:
// "The second main phase of a turn where the combat phase was skipped is
// still a postcombat main phase." Even when combat is skipped, the phase
// after PrecombatMain is still PostcombatMain — not another PrecombatMain.
//
// Uses Game.SkipNextCombatPhase() to drop the six combat steps from turn 1's
// schedule and records the steps that actually run.
func TestTurnStructureMain_PostcombatMainAfterSkippedCombat(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	tg.Schedule.BuildNextTurn()
	tg.SkipNextCombatPhase()

	var observed []core.PhaseStep
	for {
		step, ok := tg.Schedule.PopNextStep()
		if !ok {
			break
		}
		observed = append(observed, step)
		tg.Step = step
		tg.RunStepWithPriority(step)
	}

	mainPhases := []core.PhaseStep{}
	for _, s := range observed {
		if s.IsMainPhase() {
			mainPhases = append(mainPhases, s)
		}
		if s == core.BeginCombat || s == core.DeclareAttackers ||
			s == core.DeclareBlockers || s == core.FirstStrikeDamage ||
			s == core.CombatDamage || s == core.EndCombat {
			t.Errorf("CR 505.1a: combat step %v ran despite SkipNextCombatPhase()", s)
		}
	}
	if len(mainPhases) != 2 {
		t.Fatalf("CR 505.1a: expected exactly 2 main phases, got %d (%v)", len(mainPhases), observed)
	}
	if mainPhases[0] != core.PrecombatMain {
		t.Errorf("CR 505.1a: first main phase should be PrecombatMain, got %v", mainPhases[0])
	}
	if mainPhases[1] != core.PostcombatMain {
		t.Errorf("CR 505.1a: second main phase should still be PostcombatMain after a skipped combat phase, got %v", mainPhases[1])
	}
}

// TestTurnStructureMain_OnlyOneLandPerTurn verifies CR 505.6b: a player can
// play at most one land per turn. Attempting a second land play in the same
// turn must be rejected and leave the second land in hand.
func TestTurnStructureMain_OnlyOneLandPerTurn(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.AddCard(core.ZoneHand, PlayerA, "Forest", 2)

	// Drive turn 1 up to precombat main manually, without triggering the
	// harness' autoPlayLands.
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

	// First land: must succeed.
	first, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: first Forest not in hand")
	}
	if err := tg.PlayLand(playerID, first.ID()); err != nil {
		t.Fatalf("CR 505.6b: first land play failed: %v", err)
	}
	if tg.LandsPlayedThisTurn != 1 {
		t.Errorf("CR 505.6b: after first land, LandsPlayedThisTurn=%d want 1", tg.LandsPlayedThisTurn)
	}

	// Second land: must be rejected.
	second, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("second Forest should still be in hand")
	}
	err := tg.PlayLand(playerID, second.ID())
	if err == nil {
		t.Error("CR 505.6b: second land play should have failed, got nil error")
	}
	if tg.LandsPlayedThisTurn != 1 {
		t.Errorf("CR 505.6b: LandsPlayedThisTurn=%d want 1 after rejected second play", tg.LandsPlayedThisTurn)
	}
	tg.AssertPermanentCount(PlayerA, "Forest", 1)
	tg.AssertHandCount(PlayerA, "Forest", 1)
}

// TestTurnStructureMain_LandInPostcombatMain verifies CR 505.6b: a player may
// play their one land per turn in the postcombat main phase (not only
// precombat).
func TestTurnStructureMain_LandInPostcombatMain(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.AddCard(core.ZoneHand, PlayerA, "Forest")

	tg.Turn = 1
	tg.ActivePlayer = 0
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}

	// Skip through beginning and precombat main without playing the land.
	preSteps := []core.PhaseStep{
		core.Untap, core.Upkeep, core.Draw, core.PrecombatMain,
		core.BeginCombat, core.DeclareAttackers, core.DeclareBlockers,
		core.FirstStrikeDamage, core.CombatDamage, core.EndCombat,
	}
	for _, step := range preSteps {
		tg.Step = step
		tg.RunStepWithPriority(step)
	}
	if tg.LandsPlayedThisTurn != 0 {
		t.Fatalf("setup: expected no lands played yet, got %d", tg.LandsPlayedThisTurn)
	}

	tg.Step = core.PostcombatMain
	playerID := tg.getPlayerID(PlayerA)
	land, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: Forest not in hand")
	}
	if err := tg.PlayLand(playerID, land.ID()); err != nil {
		t.Fatalf("CR 505.6b: land play in postcombat main failed: %v", err)
	}
	tg.AssertPermanentCount(PlayerA, "Forest", 1)
	if tg.LandsPlayedThisTurn != 1 {
		t.Errorf("CR 505.6b: LandsPlayedThisTurn=%d want 1", tg.LandsPlayedThisTurn)
	}
}

// TestTurnStructureMain_SorceryInPrecombatMain verifies CR 505.6a: a sorcery
// can be cast during the controller's precombat main phase.
func TestTurnStructureMain_SorceryInPrecombatMain(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetLife(PlayerA, 20)
	tg.AddCard(core.ZoneHand, PlayerA, "TurnStructureMain Healing Salve Sorcery")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "TurnStructureMain Healing Salve Sorcery")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerA, 23)
	tg.AssertGraveyardCount(PlayerA, "TurnStructureMain Healing Salve Sorcery", 1)
}

// TestTurnStructureMain_SorceryInPostcombatMain verifies CR 505.6a: a sorcery
// can be cast during the controller's postcombat main phase.
func TestTurnStructureMain_SorceryInPostcombatMain(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetLife(PlayerA, 20)
	tg.AddCard(core.ZoneHand, PlayerA, "TurnStructureMain Healing Salve Sorcery")
	tg.CastSpell(1, core.PostcombatMain, PlayerA, "TurnStructureMain Healing Salve Sorcery")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerA, 23)
	tg.AssertGraveyardCount(PlayerA, "TurnStructureMain Healing Salve Sorcery", 1)
}

// TestTurnStructureMain_NonActivePlayerCannotPlayLand verifies CR 505.6b: only
// the active player can play a land during a main phase.
func TestTurnStructureMain_NonActivePlayerCannotPlayLand(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.AddCard(core.ZoneHand, PlayerB, "Forest")

	// Turn 1, PlayerA is active. Drive to PlayerA's precombat main without
	// auto-playing lands (PlayerA has no land in hand anyway).
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

	// PlayerB (non-active) attempts to play a land — must be rejected.
	playerBID := tg.getPlayerID(PlayerB)
	land, ok := findLandInHand(tg, PlayerB, "Forest")
	if !ok {
		t.Fatal("setup: PlayerB's Forest not in hand")
	}
	err := tg.PlayLand(playerBID, land.ID())
	if err == nil {
		t.Error("CR 505.6b: non-active player's land play should have failed, got nil error")
	}
	if tg.LandsPlayedThisTurn != 0 {
		t.Errorf("CR 505.6b: LandsPlayedThisTurn=%d want 0", tg.LandsPlayedThisTurn)
	}
	tg.AssertPermanentCount(PlayerB, "Forest", 0)
	tg.AssertHandCount(PlayerB, "Forest", 1)
}
