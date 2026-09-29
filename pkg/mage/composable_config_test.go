package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// DefinitionValidationCase defines a test case for ValidateCardDefinition.
type DefinitionValidationCase struct {
	Name          string
	Card          Card
	Factory       CardFactory // optional: if set, called with panic recovery (via validateFactory)
	WantErr       bool        // if true, an error is expected
	WantErrSubstr string      // if non-empty, error must contain this substring
	Inspect       func(t *testing.T)
}

// RunDefinitionValidationTable runs a table of card definition validation tests.
func RunDefinitionValidationTable(t *testing.T, cases []DefinitionValidationCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			var err error
			name := tc.Name
			if tc.Factory != nil {
				err = validateFactory(tc.Name, tc.Factory)
			} else {
				if tc.Card != nil {
					name = tc.Card.Name()
				}
				err = ValidateCardDefinition(name, tc.Card)
			}
			if !tc.WantErr && tc.WantErrSubstr == "" {
				if err != nil {
					t.Fatalf("expected valid card %q, got error: %v", name, err)
				}
				if tc.Inspect != nil {
					tc.Inspect(t)
				}
			} else {
				if err == nil {
					t.Fatalf("card %q: expected validation error, got nil", name)
				}
				if tc.WantErrSubstr != "" && !strings.Contains(err.Error(), tc.WantErrSubstr) {
					t.Fatalf("card %q: error %q does not contain %q", name, err.Error(), tc.WantErrSubstr)
				}
			}
		})
	}
}

// TriggerScenarioCase defines a test scenario for composable event triggers,
// stack placement, clone isolation, and stack resolution.
type TriggerScenarioCase struct {
	Name         string
	Setup        func(g *Game, a, b *BasePlayer)
	Action       func(g *Game, a, b *BasePlayer)
	WantTriggers int
	CopyTrigger  bool
	CopyPlayerB  bool // if true, copies trigger to player B, else to player A
	CheckClone   bool // if true, clones before resolve and asserts clone resolution isolation
	WantLifeA    *int
	WantLifeB    *int
	AssertState  func(t *testing.T, g *Game, a, b *BasePlayer)
}

// RunTriggerScenarios runs a table of trigger/event execution scenarios.
func RunTriggerScenarios(t *testing.T, cases []TriggerScenarioCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)

			if tc.Setup != nil {
				tc.Setup(g, a, b)
			}
			if tc.Action != nil {
				tc.Action(g, a, b)
			}

			g.PutTriggersOnStack()
			if g.StackSize() != tc.WantTriggers {
				t.Fatalf("stack size = %d, want %d", g.StackSize(), tc.WantTriggers)
			}

			if tc.CopyTrigger && g.StackSize() > 0 {
				targetPID := a.PlayerID()
				if tc.CopyPlayerB {
					targetPID = b.PlayerID()
				}
				if g.CopyStackObjectDirect(g.StackPeek(), targetPID, false) == nil {
					t.Fatal("could not copy trigger on stack")
				}
			}

			if tc.CheckClone {
				origLifeA, origLifeB := a.Life(), b.Life()
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != origLifeA || b.Life() != origLifeB {
					t.Fatalf("clone resolution mutated original player life: A=%d B=%d", a.Life(), b.Life())
				}
			}

			g.ResolveStack()

			if tc.WantLifeA != nil && a.Life() != *tc.WantLifeA {
				t.Fatalf("life A = %d, want %d", a.Life(), *tc.WantLifeA)
			}
			if tc.WantLifeB != nil && b.Life() != *tc.WantLifeB {
				t.Fatalf("life B = %d, want %d", b.Life(), *tc.WantLifeB)
			}
			if tc.AssertState != nil {
				tc.AssertState(t, g, a, b)
			}
		})
	}
}

// RunPhasingWatcherLifecycle tests the standard phasing lifecycle of a watcher card:
// 1. Phased out watcher must not trigger.
// 2. Phased in watcher triggers.
// 3. Pending trigger resolves correctly even if watcher changes zone/controller.
func RunPhasingWatcherLifecycle(t *testing.T, watcher Card, setup func(g *Game, a, b *BasePlayer), fireEvent func(g *Game, a, b *BasePlayer), wantLifeA, wantLifeB int) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(watcher, a.PlayerID())
	if setup != nil {
		setup(g, a, b)
	}

	phased := g.PhaseOut(watcher.ID())
	fireEvent(g, a, b)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}

	g.PhaseIn(phased)
	fireEvent(g, a, b)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("phased-in watcher triggers = %d, want 1", g.StackSize())
	}

	// Move watcher across battlefield/controller before resolving stack
	perm := g.FindPermanent(watcher.ID())
	if perm != nil {
		g.RemoveFromBattlefield(perm)
		g.PutOnBattlefield(watcher, b.PlayerID())
	}

	g.ResolveStack()
	if a.Life() != wantLifeA || b.Life() != wantLifeB {
		t.Fatalf("life totals after resolving pending trigger = A:%d B:%d (want A:%d B:%d)",
			a.Life(), b.Life(), wantLifeA, wantLifeB)
	}
}

// ensure uuid import is used
var _ = uuid.Nil
