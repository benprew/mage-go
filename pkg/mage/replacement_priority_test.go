package mage

import (
	"testing"

	"github.com/google/uuid"
)

type testMockAction struct {
	sourceID uuid.UUID
	playerID uuid.UUID
	value    int
}

func (a *testMockAction) ActionSource() uuid.UUID { return a.sourceID }
func (a *testMockAction) PlayerID() uuid.UUID     { return a.playerID }

type testPrioritizedReplacement struct {
	replacementBase
	matchFunc   func(Action) bool
	replaceFunc func(Action, *Game) Action
}

func (r *testPrioritizedReplacement) Matches(a Action, _ GameReader) bool {
	return r.matchFunc(a)
}

func (r *testPrioritizedReplacement) Replace(a Action, g *Game) Action {
	return r.replaceFunc(a, g)
}

func (r *testPrioritizedReplacement) IsActive(_ GameReader) bool { return true }

func (r *testPrioritizedReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

func TestCR616_1a_SelfReplacementOverDefault(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	var executionOrder []string

	selfRepl := &testPrioritizedReplacement{
		replacementBase: replacementBase{
			priority: ReplacementPrioritySelf,
		},
		matchFunc: func(a Action) bool {
			_, ok := a.(*testMockAction)
			return ok
		},
		replaceFunc: func(a Action, _ *Game) Action {
			executionOrder = append(executionOrder, "self")
			act := a.(*testMockAction)
			return &testMockAction{sourceID: act.sourceID, playerID: act.playerID, value: act.value + 10}
		},
	}

	defaultRepl := &testPrioritizedReplacement{
		replacementBase: replacementBase{
			priority: ReplacementPriorityDefault,
		},
		matchFunc: func(a Action) bool {
			_, ok := a.(*testMockAction)
			return ok
		},
		replaceFunc: func(a Action, _ *Game) Action {
			executionOrder = append(executionOrder, "default")
			act := a.(*testMockAction)
			return &testMockAction{sourceID: act.sourceID, playerID: act.playerID, value: act.value * 2}
		},
	}

	// Register default first, then self
	g.effects.AddReplacement(defaultRepl)
	g.effects.AddReplacement(selfRepl)

	action := &testMockAction{playerID: pA.PlayerID(), value: 5}
	res := g.effects.ApplyReplacements(action, g)

	if len(executionOrder) != 2 || executionOrder[0] != "self" || executionOrder[1] != "default" {
		t.Fatalf("expected order [self, default], got %v", executionOrder)
	}

	resultAction, ok := res.(*testMockAction)
	if !ok || resultAction.value != (5+10)*2 {
		t.Fatalf("expected value %d, got %v", (5+10)*2, resultAction)
	}
}

func TestCR616_1c_CopyOverDefault(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	var executionOrder []string

	copyRepl := &testPrioritizedReplacement{
		replacementBase: replacementBase{
			priority: ReplacementPriorityCopy,
		},
		matchFunc: func(a Action) bool {
			_, ok := a.(*testMockAction)
			return ok
		},
		replaceFunc: func(a Action, _ *Game) Action {
			executionOrder = append(executionOrder, "copy")
			return a
		},
	}

	defaultRepl := &testPrioritizedReplacement{
		replacementBase: replacementBase{
			priority: ReplacementPriorityDefault,
		},
		matchFunc: func(a Action) bool {
			_, ok := a.(*testMockAction)
			return ok
		},
		replaceFunc: func(a Action, _ *Game) Action {
			executionOrder = append(executionOrder, "default")
			return a
		},
	}

	g.effects.AddReplacement(defaultRepl)
	g.effects.AddReplacement(copyRepl)

	action := &testMockAction{playerID: pA.PlayerID(), value: 1}
	g.effects.ApplyReplacements(action, g)

	if len(executionOrder) != 2 || executionOrder[0] != "copy" || executionOrder[1] != "default" {
		t.Fatalf("expected order [copy, default], got %v", executionOrder)
	}
}

type choiceRecordingPlayer struct {
	*BasePlayer
	chosenMode int
}

func (p *choiceRecordingPlayer) ChooseMode(modes []string, title string) int {
	return p.chosenMode
}

func TestCR616_1_SameTierPlayerChoice(t *testing.T) {
	pA := &choiceRecordingPlayer{BasePlayer: NewBasePlayer("Alice"), chosenMode: 1}
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	var executionOrder []string

	repl1 := &testPrioritizedReplacement{
		replacementBase: replacementBase{
			priority: ReplacementPriorityDefault,
		},
		matchFunc: func(a Action) bool {
			_, ok := a.(*testMockAction)
			return ok
		},
		replaceFunc: func(a Action, _ *Game) Action {
			executionOrder = append(executionOrder, "repl1")
			return a
		},
	}

	repl2 := &testPrioritizedReplacement{
		replacementBase: replacementBase{
			priority: ReplacementPriorityDefault,
		},
		matchFunc: func(a Action) bool {
			_, ok := a.(*testMockAction)
			return ok
		},
		replaceFunc: func(a Action, _ *Game) Action {
			executionOrder = append(executionOrder, "repl2")
			return a
		},
	}

	g.effects.AddReplacement(repl1)
	g.effects.AddReplacement(repl2)

	action := &testMockAction{playerID: pA.PlayerID(), value: 1}
	g.effects.ApplyReplacements(action, g)

	// Since chosenMode is 1, repl2 should execute first
	if len(executionOrder) != 2 || executionOrder[0] != "repl2" || executionOrder[1] != "repl1" {
		t.Fatalf("expected order [repl2, repl1], got %v", executionOrder)
	}
}
