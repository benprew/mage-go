package search

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

func TestTopTargetCombinationsUsesAnnouncedX(t *testing.T) {
	g, player, opponent := makeGame()
	first := makePerm("First", "{1}", 1, 1, opponent.PlayerID())
	second := makePerm("Second", "{1}", 1, 1, opponent.PlayerID())
	third := makePerm("Third", "{1}", 1, 1, opponent.PlayerID())
	g.AddToBattlefield(first)
	g.AddToBattlefield(second)
	g.AddToBattlefield(third)
	spell := mage.NewSorcery("X Tap", "{X}", mage.NewMultiTargetSpell(
		[]mage.Target{mage.TargetXCreatures()},
		mage.FuncEffect("tap", mage.EffectProperties{Outcome: mage.OutcomeDetriment}, nil),
	))

	combos := topTargetCombinations(g, player.PlayerID(), spell, spell.CastTargets(), eval.TargetGeneric, 0, 2)
	if len(combos) == 0 {
		t.Fatal("expected target combinations")
	}
	for _, combo := range combos {
		if len(combo) != 2 {
			t.Fatalf("combination has %d targets, want 2", len(combo))
		}
		if combo[0] == combo[1] {
			t.Fatal("X targets must be distinct")
		}
	}
}
