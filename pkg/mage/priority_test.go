package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
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
	g.onPriority = autoPass()

	// Give each player a library so they don't deck out
	for _, p := range g.players {
		for range 60 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}
	return g
}

func TestRunStepWithPriority_Untap(t *testing.T) {
	g := newPriorityTestGame()

	// Put a tapped creature on the battlefield
	card := NewCreature("Bear", "{1}{G}", 2, 2)
	card.SetOwner(g.players[0].PlayerID())
	perm := g.PutOnBattlefield(card, g.players[0].PlayerID())
	perm.Tapped = true
	perm.RevokeBaseAttr(AttrSummonSick)

	g.RunStepWithPriority(Untap)

	if perm.Tapped {
		t.Error("expected creature to be untapped after untap step")
	}
}

func TestRunStepWithPriority_Draw(t *testing.T) {
	g := newPriorityTestGame()
	g.turn = 2 // not turn 1 so draw happens

	handBefore := len(g.players[g.activePlayer].Hand())
	g.RunStepWithPriority(Draw)
	handAfter := len(g.players[g.activePlayer].Hand())

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
	if g.step != Cleanup {
		t.Errorf("expected step Cleanup, got %v", g.step)
	}
}

func TestRunStepWithPriority_NilHandler_FullTurn(t *testing.T) {
	// Verify RunStepWithPriority works when onPriority is nil (falls back
	// to ResolveStack). Run 3 turns and confirm the game stays consistent.
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)
	for _, p := range g.players {
		for range 60 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}

	for range 3 {
		for _, step := range AllSteps() {
			g.RunStepWithPriority(step)
		}
		g.activePlayer = (g.activePlayer + 1) % 2
		g.turn++
	}

	for i, p := range g.players {
		if p.Life() != 20 {
			t.Errorf("player %d life = %d, want 20", i, p.Life())
		}
	}
}

func TestRunPriorityRound_NilHandler(t *testing.T) {
	pA := NewBasePlayer("A")
	pB := NewBasePlayer("B")
	g := NewGame(pA, pB)
	// OnPriority is nil — should fall back to ResolveStack behavior
	g.runPriorityRound(false) // should not panic
}

func TestRunPriorityRound_ActionExecution(t *testing.T) {
	g := newPriorityTestGame()
	g.step = PrecombatMain // needed for land plays

	// Add a land to hand
	land := NewLand("Forest")
	land.SetOwner(g.players[0].PlayerID())
	g.players[0].AddToHand(land)

	actionCount := 0
	g.onPriority = func(g *Game, playerIdx int, mainPhase bool) PriorityAction {
		if playerIdx == 0 && actionCount == 0 {
			actionCount++
			return PriorityAction{
				Type:   PriorityPlayLand,
				CardID: land.ID(),
			}
		}
		return PriorityAction{Type: PriorityPass}
	}

	g.runPriorityRound(true)

	if g.GetLandsPlayedThisTurn() != 1 {
		t.Errorf("expected 1 land played, got %d", g.GetLandsPlayedThisTurn())
	}
	found := false
	for _, p := range g.battlefield {
		if p.Name() == "Forest" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Forest on battlefield")
	}
}
