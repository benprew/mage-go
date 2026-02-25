package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"testing"
)

// autoPass is a PriorityHandler that always passes.
func autoPass() PriorityHandler {
	return func(g *Game, playerIdx int, mainPhase bool) PriorityAction {
		return PriorityAction{Type: PriorityPass}
	}
}

func newPriorityTestGame() *Game {
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)
	g.OnPriority = autoPass()

	// Give each player a library so they don't deck out
	for _, p := range g.Players {
		for i := 0; i < 60; i++ {
			p.AddToLibrary(NewLand("Plains"))
		}
	}
	return g
}

func TestRunStepWithPriority_Untap(t *testing.T) {
	g := newPriorityTestGame()

	// Put a tapped creature on the battlefield
	card := NewCreature("Bear", "{1}{G}", 2, 2)
	card.SetOwner(g.Players[0].PlayerID())
	perm := g.PutOnBattlefield(card, g.Players[0].PlayerID())
	perm.Tapped = true
	perm.RevokeBaseAttr(AttrSummonSick)

	g.RunStepWithPriority(Untap)

	if perm.Tapped {
		t.Error("expected creature to be untapped after untap step")
	}
}

func TestRunStepWithPriority_Draw(t *testing.T) {
	g := newPriorityTestGame()
	g.Turn = 2 // not turn 1 so draw happens

	handBefore := len(g.Players[g.ActivePlayer].Hand())
	g.RunStepWithPriority(Draw)
	handAfter := len(g.Players[g.ActivePlayer].Hand())

	if handAfter != handBefore+1 {
		t.Errorf("expected hand size %d after draw, got %d", handBefore+1, handAfter)
	}
}

func TestRunStepWithPriority_FullTurn(t *testing.T) {
	g := newPriorityTestGame()

	// Run a full turn using RunStepWithPriority
	for _, step := range AllSteps() {
		g.RunStepWithPriority(step)
		if g.IsGameOver() {
			t.Fatal("game ended unexpectedly")
		}
	}

	// Verify the turn completed normally
	if g.Step != Cleanup {
		t.Errorf("expected step Cleanup, got %v", g.Step)
	}
}

func TestRunStepWithPriority_MatchesRunStep(t *testing.T) {
	// Run the same setup through RunStep and RunStepWithPriority
	// and verify life totals match (basic smoke test)

	makeGame := func() *Game {
		pA := NewBasePlayer("A")
		pB := NewBasePlayer("B")
		g := NewGame(pA, pB)
		for _, p := range g.Players {
			for i := 0; i < 60; i++ {
				p.AddToLibrary(NewLand("Plains"))
			}
		}
		return g
	}

	g1 := makeGame()
	g2 := makeGame()
	g2.OnPriority = autoPass()

	// Run 3 turns through each
	for turn := 0; turn < 3; turn++ {
		for _, step := range AllSteps() {
			g1.RunStep(step)
		}
		g1.ActivePlayer = (g1.ActivePlayer + 1) % 2
		g1.Turn++

		for _, step := range AllSteps() {
			g2.RunStepWithPriority(step)
		}
		g2.ActivePlayer = (g2.ActivePlayer + 1) % 2
		g2.Turn++
	}

	for i := range g1.Players {
		life1 := g1.Players[i].Life()
		life2 := g2.Players[i].Life()
		if life1 != life2 {
			t.Errorf("player %d life mismatch: RunStep=%d, RunStepWithPriority=%d", i, life1, life2)
		}
		hand1 := len(g1.Players[i].Hand())
		hand2 := len(g2.Players[i].Hand())
		if hand1 != hand2 {
			t.Errorf("player %d hand size mismatch: RunStep=%d, RunStepWithPriority=%d", i, hand1, hand2)
		}
	}
}

func TestRunPriorityRound_NilHandler(t *testing.T) {
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)
	// OnPriority is nil — should fall back to ResolveStack behavior
	g.RunPriorityRound(false) // should not panic
}

func TestRunPriorityRound_ActionExecution(t *testing.T) {
	g := newPriorityTestGame()
	g.Step = PrecombatMain // needed for land plays

	// Add a land to hand
	land := NewLand("Forest")
	land.SetOwner(g.Players[0].PlayerID())
	g.Players[0].AddToHand(land)

	actionCount := 0
	g.OnPriority = func(g *Game, playerIdx int, mainPhase bool) PriorityAction {
		if playerIdx == 0 && actionCount == 0 {
			actionCount++
			return PriorityAction{
				Type:   PriorityPlayLand,
				CardID: land.ID(),
			}
		}
		return PriorityAction{Type: PriorityPass}
	}

	g.RunPriorityRound(true)

	if g.LandsPlayedThisTurn != 1 {
		t.Errorf("expected 1 land played, got %d", g.LandsPlayedThisTurn)
	}
	found := false
	for _, p := range g.Battlefield {
		if p.Name() == "Forest" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Forest on battlefield")
	}
}
