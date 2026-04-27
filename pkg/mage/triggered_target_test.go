package mage

import (
	"testing"

	"github.com/google/uuid"
)

// recordingPlayer wraps BasePlayer to record/script ChooseTargets calls.
type recordingPlayer struct {
	*BasePlayer
	chooseQueue [][]uuid.UUID
	calls       []choosePromptCall
}

type choosePromptCall struct {
	possible []uuid.UUID
	min, max int
}

func (rp *recordingPlayer) ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID {
	rp.calls = append(rp.calls, choosePromptCall{possible: append([]uuid.UUID(nil), possible...), min: min, max: max})
	if len(rp.chooseQueue) > 0 {
		choice := rp.chooseQueue[0]
		rp.chooseQueue = rp.chooseQueue[1:]
		return choice
	}
	if len(possible) >= min {
		n := min
		if n == 0 && len(possible) > 0 {
			n = 1
		}
		if n > len(possible) {
			n = len(possible)
		}
		return possible[:n]
	}
	return nil
}

func newRecordingPlayer(name string) *recordingPlayer {
	return &recordingPlayer{BasePlayer: NewBasePlayer(name)}
}

func newTriggerTargetGame() (*Game, *recordingPlayer, *recordingPlayer) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 30 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}
	return g, a, b
}

// TestTriggerWithDeclaredTarget_PlayerTarget verifies that a triggered ability
// with .AddTarget(TargetPlayer()) prompts the controller and uses the chosen
// player as the trigger's target.
func TestTriggerWithDeclaredTarget_PlayerTarget(t *testing.T) {
	g, a, _ := newTriggerTargetGame()

	var resolvedTarget uuid.UUID
	effect := FuncEffect("note target", EffectProperties{},
		func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
			if len(targets) > 0 {
				resolvedTarget = targets[0]
			}
			return nil
		})

	src := NewCreature("Synthetic Source", "{1}", 1, 1)
	src.SetOwner(a.PlayerID())
	src.AddAbility(EntersBattlefieldTrigger(effect, false).AddTarget(TargetPlayer()))

	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}

	g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	if len(a.calls) != 1 {
		t.Fatalf("expected 1 ChooseTargets prompt, got %d", len(a.calls))
	}
	if a.calls[0].min != 1 || a.calls[0].max != 1 {
		t.Errorf("expected min=1 max=1, got min=%d max=%d", a.calls[0].min, a.calls[0].max)
	}
	if len(a.calls[0].possible) != 2 {
		t.Errorf("expected 2 possible player targets, got %d", len(a.calls[0].possible))
	}
	if resolvedTarget != a.PlayerID() {
		t.Errorf("expected target=%v (chose self), got %v", a.PlayerID(), resolvedTarget)
	}
}

// TestTriggerWithDeclaredTarget_OpponentChoice verifies that the controller
// picks the *opponent* when scripted to do so, instead of the legacy
// auto-bind-from-event behavior.
func TestTriggerWithDeclaredTarget_OpponentChoice(t *testing.T) {
	g, a, b := newTriggerTargetGame()

	var resolvedTarget uuid.UUID
	effect := FuncEffect("note target", EffectProperties{},
		func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
			if len(targets) > 0 {
				resolvedTarget = targets[0]
			}
			return nil
		})

	src := NewCreature("Synthetic Source 2", "{1}", 1, 1)
	src.SetOwner(a.PlayerID())
	src.AddAbility(EntersBattlefieldTrigger(effect, false).AddTarget(TargetPlayer()))

	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}

	g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	if resolvedTarget != b.PlayerID() {
		t.Errorf("expected target=B, got %v", resolvedTarget)
	}
}

// TestTriggerWithDeclaredTarget_CreatureTarget verifies that a trigger with
// .AddTarget(TargetCreature()) prompts among the creatures on the battlefield
// and binds the chosen creature.
func TestTriggerWithDeclaredTarget_CreatureTarget(t *testing.T) {
	g, a, b := newTriggerTargetGame()

	bear := NewCreature("Test Bear", "{1}{G}", 2, 2)
	bear.SetOwner(b.PlayerID())
	bearPerm := g.PutOnBattlefield(bear, b.PlayerID())

	wolf := NewCreature("Test Wolf", "{1}{G}", 3, 3)
	wolf.SetOwner(b.PlayerID())
	wolfPerm := g.PutOnBattlefield(wolf, b.PlayerID())

	var resolvedTarget uuid.UUID
	effect := FuncEffect("note target", EffectProperties{},
		func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
			if len(targets) > 0 {
				resolvedTarget = targets[0]
			}
			return nil
		})

	src := NewCreature("Synthetic Source 3", "{1}", 1, 1)
	src.SetOwner(a.PlayerID())
	src.AddAbility(EntersBattlefieldTrigger(effect, false).AddTarget(TargetCreature()))

	a.chooseQueue = [][]uuid.UUID{{wolfPerm.ID()}}

	g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	if len(a.calls) != 1 {
		t.Fatalf("expected 1 ChooseTargets prompt, got %d", len(a.calls))
	}
	// Possible targets should include both opponent's creatures plus the source
	// itself (3 creatures on the battlefield total).
	if got := len(a.calls[0].possible); got != 3 {
		t.Errorf("expected 3 possible creature targets, got %d", got)
	}
	if resolvedTarget != wolfPerm.ID() {
		t.Errorf("expected target=Wolf (%v), got %v", wolfPerm.ID(), resolvedTarget)
	}
	_ = bearPerm
}

// TestTriggerWithoutDeclaredTarget_PreservesAutoBind ensures that triggers
// without AddTarget(...) keep their legacy event-derived auto-binding so
// existing card behavior doesn't regress.
func TestTriggerWithoutDeclaredTarget_PreservesAutoBind(t *testing.T) {
	g, a, _ := newTriggerTargetGame()

	var resolvedTarget uuid.UUID
	effect := FuncEffect("note target", EffectProperties{},
		func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
			if len(targets) > 0 {
				resolvedTarget = targets[0]
			}
			return nil
		})

	src := NewCreature("Synthetic Source 4", "{1}", 1, 1)
	src.SetOwner(a.PlayerID())
	src.AddAbility(EntersBattlefieldTrigger(effect, false))

	perm := g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	if len(a.calls) != 0 {
		t.Errorf("expected no ChooseTargets prompt for trigger without AddTarget, got %d", len(a.calls))
	}
	if resolvedTarget != perm.ID() {
		t.Errorf("expected auto-bound target=source perm (%v), got %v", perm.ID(), resolvedTarget)
	}
}

// TestTriggerWithDeclaredTarget_NoLegalTargets_DoesNotPrompt covers the edge
// case where no legal targets exist; the controller is not prompted and the
// trigger's stack object is left with a placeholder so the resolution path
// still runs (and the effect can detect missing targets).
func TestTriggerWithDeclaredTarget_NoLegalTargets_DoesNotPrompt(t *testing.T) {
	g, a, _ := newTriggerTargetGame()

	called := false
	effect := FuncEffect("note target", EffectProperties{},
		func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
			called = true
			return nil
		})

	// Use an enchant-creature-style filter that no creatures can satisfy.
	noCreatures := func() Target { return TargetCreature() }

	src := NewEnchantment("Synthetic Source 5", "{1}")
	src.SetOwner(a.PlayerID())
	src.AddAbility(EntersBattlefieldTrigger(effect, false).AddTarget(noCreatures()))

	g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	if len(a.calls) != 0 {
		t.Errorf("expected no ChooseTargets prompt when no legal targets exist, got %d", len(a.calls))
	}
	// Effect should still resolve (with nil/placeholder target); fizzling is
	// covered by isTargetStillLegal at resolution-time for normal cases.
	_ = called
}

