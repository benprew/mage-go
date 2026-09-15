package search

import (
	"testing"
	"time"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// buildMeasureState builds a moderately complex mid-game state with both
// players holding several cards, mana available, and creatures on the board.
// The transposition table pays off more at higher branching factors, so a
// dense state is a better showcase than a starting-turn position.
func buildMeasureState() (*mage.Game, *mage.BasePlayer, *mage.BasePlayer) {
	g, pa, pb := makeGame()
	pa.SetLife(15)
	pb.SetLife(11)

	addLands(g, pa, "Mountain", 4)
	addLands(g, pa, "Forest", 2)
	addLands(g, pb, "Plains", 3)
	addLands(g, pb, "Island", 2)

	g.AddToBattlefield(makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID()))
	g.AddToBattlefield(makePerm("Hill Giant", "{3}{R}", 3, 3, pa.PlayerID()))
	g.AddToBattlefield(makePerm("Savannah Lions", "{W}", 2, 1, pb.PlayerID()))
	g.AddToBattlefield(makePerm("Wall of Swords", "{3}{W}", 3, 5, pb.PlayerID()))

	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(pa.PlayerID())
		pa.AddToHand(c)
	}
	for range 2 {
		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))))
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)
	}

	g.SetStep(core.PrecombatMain)
	return g, pa, pb
}

// BenchmarkMeasure_TTImpact runs identical searches with and without the
// transposition table and reports nodes, TT activity, and wall-clock time.
// Run via: go test -bench=BenchmarkMeasure_TTImpact -v ./pkg/mage/interactive/ai/search
func BenchmarkMeasure_TTImpact(b *testing.B) {
	cfg := Config{
		MaxDepth:  6,
		MaxNodes:  1 << 30,
		TimeLimit: 60 * time.Second,
	}

	b.Run("NoTT", func(b *testing.B) {
		var lastNodes uint64
		for i := 0; i < b.N; i++ {
			g, pa, _ := buildMeasureState()
			noTT := &Strategy{
				Config:    cfg,
				Evaluator: eval.DefaultEvaluator,
			}
			noTT.PriorityAction(pa, g, 0, true)
			lastNodes = noTT.LastNodes
		}
		b.ReportMetric(float64(lastNodes), "nodes/op")
	})

	b.Run("WithTT", func(b *testing.B) {
		var lastNodes uint64
		for i := 0; i < b.N; i++ {
			g, pa, _ := buildMeasureState()
			withTT := &Strategy{
				Config:    cfg,
				Evaluator: eval.DefaultEvaluator,
				tt:        NewTranspositionTable(DefaultTTSizeMB),
				zobrist:   NewZobristTables(),
			}
			withTT.PriorityAction(pa, g, 0, true)
			lastNodes = withTT.LastNodes
		}
		b.ReportMetric(float64(lastNodes), "nodes/op")
	})
}
