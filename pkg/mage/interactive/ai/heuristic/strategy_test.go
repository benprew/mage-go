package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
)

// ── Adaptive ────────────────────────────────────────────────────────────────

func TestAdaptive_AheadUsesAggressive(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)

	adaptive := &Adaptive{
		Aggressive: &Strategy{Personality: ai.AggroPersonality},
		Defensive:  &Strategy{Personality: ai.ControlPersonality},
	}
	got := adaptive.active(pa, g)
	if got != adaptive.Aggressive {
		t.Error("adaptive should use Aggressive when ahead")
	}
}

func TestAdaptive_BehindUsesDefensive(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(5)
	pb.SetLife(20)
	oppCreature := makePerm("Giant", "{3}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	adaptive := &Adaptive{
		Aggressive: &Strategy{Personality: ai.AggroPersonality},
		Defensive:  &Strategy{Personality: ai.ControlPersonality},
	}
	got := adaptive.active(pa, g)
	if got != adaptive.Defensive {
		t.Error("adaptive should use Defensive when behind")
	}
}

func TestAdaptive_WeightedPresets(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)

	adaptive := &Adaptive{
		Aggressive: New(ai.AggroWeighted),
		Defensive:  New(ai.ControlWeighted),
	}
	got := adaptive.active(pa, g)
	if got != adaptive.Aggressive {
		t.Error("adaptive should use Aggressive when ahead")
	}

	pa.SetLife(5)
	pb.SetLife(20)
	oppCreature := makePerm("Giant", "{3}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	got = adaptive.active(pa, g)
	if got != adaptive.Defensive {
		t.Error("adaptive should use Defensive when behind")
	}
}

// ── Sequential ──────────────────────────────────────────────────────────────

type passStrategy struct{}

func (s *passStrategy) PriorityAction(_ mage.Player, _ *mage.Game, _ int, _ bool) interactive.PriorityAction {
	return interactive.PriorityAction{Type: interactive.ActionPass}
}
func (s *passStrategy) Attackers(_ mage.Player, _ *mage.Game) []uuid.UUID           { return nil }
func (s *passStrategy) Blockers(_ mage.Player, _ *mage.Game) []mage.BlockAssignment { return nil }

type fixedActionStrategy struct {
	action interactive.PriorityAction
}

func (s *fixedActionStrategy) PriorityAction(_ mage.Player, _ *mage.Game, _ int, _ bool) interactive.PriorityAction {
	return s.action
}
func (s *fixedActionStrategy) Attackers(_ mage.Player, _ *mage.Game) []uuid.UUID { return nil }
func (s *fixedActionStrategy) Blockers(_ mage.Player, _ *mage.Game) []mage.BlockAssignment {
	return nil
}

func TestSequential_FirstNonPassWins(t *testing.T) {
	g, pa, _ := makeGame()
	landAction := interactive.PriorityAction{Type: interactive.ActionPlayLand, CardName: "Forest"}

	seq := &Sequential{
		Strategies: []ai.AIStrategy{
			&passStrategy{},
			&fixedActionStrategy{action: landAction},
		},
	}
	got := seq.PriorityAction(pa, g, 0, true)
	if got.Type != interactive.ActionPlayLand {
		t.Errorf("expected first non-pass action, got %v", got.Type)
	}
}

func TestSequential_AllPassReturnsPass(t *testing.T) {
	g, pa, _ := makeGame()
	seq := &Sequential{
		Strategies: []ai.AIStrategy{
			&passStrategy{},
			&passStrategy{},
		},
	}
	got := seq.PriorityAction(pa, g, 0, true)
	if got.Type != interactive.ActionPass {
		t.Errorf("expected pass when all strategies pass, got %v", got.Type)
	}
}

func TestSequential_Attackers(t *testing.T) {
	g, pa, _ := makeGame()
	seq := &Sequential{
		Strategies: []ai.AIStrategy{
			&passStrategy{},
			&passStrategy{},
		},
	}
	atks := seq.Attackers(pa, g)
	if len(atks) != 0 {
		t.Errorf("expected no attackers, got %d", len(atks))
	}
}

func TestSequential_Blockers(t *testing.T) {
	g, pa, _ := makeGame()
	seq := &Sequential{
		Strategies: []ai.AIStrategy{
			&passStrategy{},
		},
	}
	blks := seq.Blockers(pa, g)
	if len(blks) != 0 {
		t.Errorf("expected no blockers, got %d", len(blks))
	}
}
