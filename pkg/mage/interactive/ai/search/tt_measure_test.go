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

// TestMeasure_TTImpact runs identical searches with and without the
// transposition table and reports nodes, TT activity, and wall-clock time.
// It is not an assertion-driven test; it exists to make A/B measurement
// trivial via `go test -run Measure -v`.
func TestMeasure_TTImpact(t *testing.T) {
	// Generous budget so both variants complete the full iterative deepening
	// — the point is to compare work-to-the-same-answer, not who stopped first.
	cfg := Config{
		MaxDepth:  6,
		MaxNodes:  1 << 30,
		TimeLimit: 60 * time.Second,
	}

	// ── Baseline: no TT ──────────────────────────────────────────────────
	g1, pa1, _ := buildMeasureState()
	noTT := &Strategy{
		Config:    cfg,
		Evaluator: eval.DefaultEvaluator,
	}
	start := time.Now()
	actionNoTT := noTT.PriorityAction(pa1, g1, 0, true)
	elapsedNoTT := time.Since(start)
	nodesNoTT := noTT.LastNodes

	// ── TT-enabled ───────────────────────────────────────────────────────
	g2, pa2, _ := buildMeasureState()
	withTT := &Strategy{
		Config:    cfg,
		Evaluator: eval.DefaultEvaluator,
		tt:        NewTranspositionTable(DefaultTTSizeMB),
		zobrist:   NewZobristTables(),
	}
	start = time.Now()
	actionTT := withTT.PriorityAction(pa2, g2, 0, true)
	elapsedTT := time.Since(start)
	nodesTT := withTT.LastNodes

	t.Logf("── TT impact ──")
	t.Logf("no TT : nodes=%d  time=%v  action=%v %s",
		nodesNoTT, elapsedNoTT, actionNoTT.Type, actionNoTT.CardName)
	t.Logf("with TT: nodes=%d  time=%v  action=%v %s",
		nodesTT, elapsedTT, actionTT.Type, actionTT.CardName)
	t.Logf("TT hits=%d stores=%d hit rate=%.1f%%",
		withTT.TTHits, withTT.TTStores,
		100*float64(withTT.TTHits)/float64(withTT.TTHits+withTT.TTStores+1))
	if nodesNoTT > 0 {
		t.Logf("node reduction: %.1f%%  (%d → %d)",
			100*float64(int64(nodesNoTT)-int64(nodesTT))/float64(nodesNoTT),
			nodesNoTT, nodesTT)
	}
	if elapsedNoTT > 0 {
		t.Logf("time reduction: %.1f%%  (%v → %v)",
			100*float64(elapsedNoTT-elapsedTT)/float64(elapsedNoTT),
			elapsedNoTT, elapsedTT)
	}
}
