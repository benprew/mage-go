package mage

import (
	"slices"
	"testing"
	"uuid"

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

func TestRunPriorityRound_ActingPlayerRetainsPriority(t *testing.T) {
	g, _, b := invariantTestGame()
	spell := NewInstant("Response", "", NewSpellAbility())
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)

	var order []int
	g.SetOnPriority(func(_ *Game, playerIdx int, _ bool) PriorityAction {
		order = append(order, playerIdx)
		if len(order) == 2 {
			return PriorityAction{Type: PriorityCastSpell, CardID: spell.ID()}
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Upkeep)
	if want := []int{0, 1, 1, 0, 0, 1}; !slices.Equal(order, want) {
		t.Errorf("priority order = %v, want %v", order, want)
	}
}

func TestRunPriorityRound_NoPriorityAfterLoss(t *testing.T) {
	g, a, _ := invariantTestGame()
	g.SetTurn(2)
	a.SetLibrary(nil)
	g.SetOnPriority(func(_ *Game, playerIdx int, _ bool) PriorityAction {
		t.Errorf("player %d received priority after deck-out", playerIdx)
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Draw)
	if !g.IsGameOver() {
		t.Error("expected the game to end")
	}
}

type priorityTestAttacker struct {
	*BasePlayer
	attackers []uuid.UUID
}

func (p *priorityTestAttacker) DeclareAttackers(_ *Game) []uuid.UUID {
	return p.attackers
}

func TestRunPriorityRound_AttackersUpdateContinuousEffects(t *testing.T) {
	a := &priorityTestAttacker{BasePlayer: NewBasePlayer("Alice")}
	g := NewGame(a, NewBasePlayer("Bob"))
	addInvariantPermanent(g, a.BasePlayer, NewEnchantment("Castle", "{3}{W}",
		WithStaticAbility(BoostControlledCreatures(0, 2, IsUntapped))))
	attacker := addInvariantPermanent(g, a.BasePlayer, NewCreature("Pegasus", "", 1, 1, WithKeyword(Haste)))
	a.attackers = []uuid.UUID{attacker.ID()}
	g.SetOnPriority(func(g *Game, _ int, _ bool) PriorityAction {
		if got := attacker.CurrentToughness(g); got != 1 || !attacker.Tapped {
			t.Errorf("attacking creature tapped = %t, toughness = %d, want true and 1", attacker.Tapped, got)
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(DeclareAttackers)
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
	g.SetTurn(2) // not turn 1 so draw happens

	handBefore := len(g.ActivePlayer().Hand())
	g.RunStepWithPriority(Draw)
	handAfter := len(g.ActivePlayer().Hand())

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
	if g.GetStep() != Cleanup {
		t.Errorf("expected step Cleanup, got %v", g.GetStep())
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
		g.SetActivePlayerIndex((g.ActivePlayerIndex() + 1) % 2)
		g.SetTurn(g.CurrentTurn() + 1)
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
	g.SetStep(PrecombatMain) // needed for land plays

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
	for _, p := range g.zones.battlefield {
		if p.Name() == "Forest" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Forest on battlefield")
	}
}
