package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// passiveAttackRegression is a negative control for the simulation harness.
// It does not attack unless the current board has lethal damage.
type passiveAttackRegression struct {
	ai.AIStrategy
}

func newPassiveAttackRegression(wp ai.WeightedPersonality) ai.AIStrategy {
	return &passiveAttackRegression{AIStrategy: New(wp)}
}

func (s *passiveAttackRegression) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	lethal := eval.CalculateLethal(g, p.PlayerID())
	if lethal.IHaveLethal {
		return lethal.LethalAttackers
	}
	return nil
}

func TestPassiveAttackRegression_SkipsProfitableNonlethalAttack(t *testing.T) {
	g, pa, _ := makeGame()
	attacker := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(attacker)

	strategy := newPassiveAttackRegression(ai.MidrangeWeighted)
	if attackers := strategy.Attackers(pa, g); len(attackers) != 0 {
		t.Fatalf("expected the negative control to skip a nonlethal attack, got %v", attackers)
	}
}

func TestSimulation_PassiveAttackRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1k simulation in short mode")
	}
	run1kGames(t, "Passive attack regression vs baseline", newPassiveAttackRegression, newBaselineStrategy)
}
