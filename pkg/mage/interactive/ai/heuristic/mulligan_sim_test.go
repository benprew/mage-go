package heuristic

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/benprew/mage-go/internal/tui"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestSimulation_MulliganPolicySetup(t *testing.T) {
	deck := []tui.DeckEntry{{Name: "Plains", Count: 20}}
	for _, keep := range []bool{false, true} {
		player := ai.NewAIPlayer("Player", New(ai.MidrangeWeighted))
		opening := simulationOpening{life: 10}
		if keep {
			opening.mulligan = func(*ai.AIPlayer) bool { return false }
		}
		prepareSimulationPlayer(player, deck, 42, opening)
		want := 5
		if keep {
			want = 7
		}
		if len(player.Hand()) != want || player.Life() != 10 || player.StartingLife() != 10 {
			t.Fatalf("keep=%v: hand=%d life=%d starting=%d", keep, len(player.Hand()), player.Life(), player.StartingLife())
		}
	}
}

func TestSimulation_MulligansVsKeep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full AI comparison in short mode")
	}
	for _, life := range []int{20, 10} {
		t.Run(fmt.Sprintf("Life%d", life), func(t *testing.T) {
			factory := func(w ai.WeightedPersonality) ai.AIStrategy { return New(w) }
			run1kGames(t, fmt.Sprintf("Existing mulligans vs keep all; %d life", life), factory, factory,
				simulationOpening{life: life},
				simulationOpening{life: life, mulligan: func(*ai.AIPlayer) bool { return false }})
		})
	}
}

func TestSimulation_MulliganOptionsFollowStrategy(t *testing.T) {
	t.Setenv("SIM_GAMES", "2")
	t.Setenv("SIM_MAX_TURNS", "1")
	var checks atomic.Int32
	factory := func(life int) func(ai.WeightedPersonality) ai.AIStrategy {
		return func(ai.WeightedPersonality) ai.AIStrategy {
			return &mirrorCheckingStrategy{t: t, onOpening: func(p mage.Player) {
				checks.Add(1)
				if p.Life() != life || p.StartingLife() != life {
					t.Errorf("strategy with %d life got %d, starting %d", life, p.Life(), p.StartingLife())
				}
			}}
		}
	}
	run1kGames(t, "Opening options follow seat swaps", factory(10), factory(20),
		simulationOpening{life: 10}, simulationOpening{life: 20})
	if checks.Load() != 4 {
		t.Fatalf("expected four opening checks, got %d", checks.Load())
	}
}
