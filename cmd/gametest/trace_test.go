package main

import (
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/internal/tui"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
)

func TestPredictedOptionsResolveWithoutChangingGame(t *testing.T) {
	a := ai.NewAIPlayer("Alice", heuristic.New(ai.MidrangeWeighted))
	b := ai.NewAIPlayer("Bob", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := mage.NewCreature("Free creature", "{0}", 2, 1)
	a.AddToHand(card)
	options := []interactive.ActionOption{
		{Type: interactive.ActionCastSpell, CardID: card.ID(), Label: "Cast Free creature {0}"},
		{Type: interactive.ActionPass, Label: "Pass"},
	}
	branch := g.Clone()
	if err := branch.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	branch.ResolveStack()
	branch.CheckStateBasedActions()
	want := fmt.Sprintf("1. Cast Free creature {0} (%+d/%+d)", a.Strategy.(*heuristic.Strategy).PositionValue(branch, a.PlayerID()), b.Strategy.(*heuristic.Strategy).PositionValue(branch, b.PlayerID()))
	var out strings.Builder
	printPredictedOptions(&out, g, a, options)
	if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "2. Pass (") {
		t.Fatalf("unexpected predictions:\n%s", out.String())
	}
	if len(a.Hand()) != 1 || len(g.AllBattlefield()) != 0 || len(g.StackObjects()) != 0 {
		t.Fatal("prediction changed the live game")
	}
}

func TestPredictionTargetsAndPassResolution(t *testing.T) {
	a := ai.NewAIPlayer("Alice", heuristic.New(ai.MidrangeWeighted))
	b := ai.NewAIPlayer("Bob", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	bolt, err := mage.CreateCard("Lightning Bolt")
	if err != nil {
		t.Fatal(err)
	}
	bolt.SetOwner(a.PlayerID())
	a.AddToHand(bolt)
	a.ManaPool().Add(core.Red, 1)
	b.SetLife(3)
	strategy := a.Strategy.(*heuristic.Strategy)
	branch, ok := strategy.PredictOption(g, a.PlayerID(), interactive.ActionOption{Type: interactive.ActionCastSpell, CardID: bolt.ID()})
	if !ok || branch.GetPlayer(b.PlayerID()).Life() != 0 {
		t.Fatal("prediction did not target Bob with lethal damage")
	}
	if b.Life() != 3 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("prediction changed life, hand, or mana in the live game")
	}
	if err := g.CastSpellByID(a.PlayerID(), bolt.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	branch, ok = strategy.PredictOption(g, b.PlayerID(), interactive.ActionOption{Type: interactive.ActionPass})
	if !ok || branch.GetPlayer(b.PlayerID()).Life() != 0 {
		t.Fatal("pass prediction did not resolve the stack")
	}
	if b.Life() != 3 || len(g.StackObjects()) != 1 {
		t.Fatal("pass prediction changed the live stack")
	}
	var out strings.Builder
	printPredictedOptions(&out, g, a, []interactive.ActionOption{{Type: interactive.ActionCastSpell, CardID: bolt.ID(), Label: "Unavailable spell"}})
	if !strings.Contains(out.String(), "(prediction unavailable)") {
		t.Fatalf("invalid option has a score: %s", out.String())
	}
}

func TestPrintDecisionStateShowsBothPlayers(t *testing.T) {
	a := ai.NewAIPlayer("Alice", heuristic.New(ai.MidrangeWeighted))
	b := ai.NewAIPlayer("Bob", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(a, b)
	a.SetLife(4)
	b.SetLife(3)
	a.AddToHand(mage.NewCreature("Hand creature", "{G}", 1, 1))
	g.AddToBattlefield(mage.NewPermanent(mage.NewCreature("Flyer", "{U}", 1, 1, mage.WithKeyword(core.Flying)), b.PlayerID()))
	var out strings.Builder
	printPositionScores(&out, g)
	printBoardState(&out, g)
	for _, want := range []string{"Alice eval:", "Bob eval:", "Alice: 4 life", "Bob: 3 life", "Hand creature", "Flyer 1/1", "Flying", "Alice's battlefield: (empty)", "hand: [(empty)]"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
}

func TestPositionScoresOmitBoardAndHand(t *testing.T) {
	a := ai.NewAIPlayer("Alice", heuristic.New(ai.MidrangeWeighted))
	b := ai.NewAIPlayer("Bob", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(a, b)
	var out strings.Builder
	printPositionScores(&out, g)
	if !strings.Contains(out.String(), "Alice eval:") || !strings.Contains(out.String(), "Bob eval:") {
		t.Fatal("missing player scores")
	}
	if strings.Contains(out.String(), "battlefield") || strings.Contains(out.String(), "hand:") {
		t.Fatalf("score output includes board or hand:\n%s", out.String())
	}
}

func TestDecisionTraceShowsBlocksAndBothScores(t *testing.T) {
	a := ai.NewAIPlayer("Alice", heuristic.New(ai.MidrangeWeighted))
	b := ai.NewAIPlayer("Bob", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(a, b)
	attacker := mage.NewPermanent(mage.NewCreature("Attacker", "{G}", 2, 2), a.PlayerID())
	blocker := mage.NewPermanent(mage.NewCreature("Blocker", "{G}", 4, 4), b.PlayerID())
	g.AddToBattlefield(attacker, blocker)
	g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
	g.SetStep(core.DeclareBlockers)
	var out strings.Builder
	strategy := &decisionTraceStrategy{AIStrategy: b.Strategy, out: &out}
	blocks := strategy.Blockers(b, g)
	if len(blocks) != 1 || blocks[0].BlockerID != blocker.ID() || blocks[0].AttackerID != attacker.ID() {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	for _, want := range []string{"Bob declares blocks", "Blocker 4/4 blocks Attacker 2/2", "Alice eval:", "Bob eval:"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
}

func TestDecisionTraceOmitsEmptyCombat(t *testing.T) {
	a := ai.NewAIPlayer("Alice", heuristic.New(ai.MidrangeWeighted))
	b := ai.NewAIPlayer("Bob", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(a, b)
	var out strings.Builder
	strategy := &decisionTraceStrategy{AIStrategy: a.Strategy, out: &out}
	strategy.Attackers(a, g)
	strategy.Blockers(b, g)
	if out.Len() != 0 {
		t.Fatalf("empty combat produced output:\n%s", out.String())
	}
}

func TestSeededGameDeterminism(t *testing.T) {
	runSeeded := func(seed uint64) string {
		deckA := tui.Archetypes[0]
		deckB := tui.Archetypes[1]
		wpA := resolvePersonality("auto", deckA.Entries)
		wpB := resolvePersonality("auto", deckB.Entries)
		playerA := createAI("Alice", wpA, "heuristic")
		playerB := createAI("Bob", wpB, "heuristic")

		for _, c := range tui.BuildDeckUnshuffled(deckA.Entries, playerA.PlayerID()) {
			playerA.AddToLibrary(c)
		}
		for _, c := range tui.BuildDeckUnshuffled(deckB.Entries, playerB.PlayerID()) {
			playerB.AddToLibrary(c)
		}

		g := mage.NewGame(playerA, playerB)
		g.SetSeed(seed)
		playerA.ShuffleLibrary()
		playerB.ShuffleLibrary()
		tui.DrawOpeningHand(playerA)
		tui.DrawOpeningHand(playerB)
		ai.MulliganAI(playerA)
		ai.MulliganAI(playerB)

		openA := handStr(playerA.Hand())
		openB := handStr(playerB.Hand())
		runGame(g, 10, true, false)

		bf := g.AllBattlefield()
		board := make([]string, 0, len(bf))
		for _, perm := range bf {
			board = append(board, fmt.Sprintf("%s:%v:%d", perm.Name(), perm.Tapped, perm.Damage))
		}
		return fmt.Sprintf("openA=%s\nopenB=%s\nturn=%d\nlife=%d/%d\nhands=%s|%s\ngy=%d/%d\nlib=%d/%d\nboard=%s",
			openA, openB, g.CurrentTurn(),
			playerA.Life(), playerB.Life(),
			handStr(playerA.Hand()), handStr(playerB.Hand()),
			len(playerA.Graveyard()), len(playerB.Graveyard()),
			len(playerA.Library()), len(playerB.Library()),
			strings.Join(board, ","),
		)
	}

	first := runSeeded(20261121)
	second := runSeeded(20261121)
	if first != second {
		t.Fatalf("seeded game diverged:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", first, second)
	}
}
