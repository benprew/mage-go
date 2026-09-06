package heuristic

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/benprew/mage-go/cards"
	"github.com/benprew/mage-go/internal/tui"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func convertActionToPriority(action interactive.PriorityAction) mage.PriorityAction {
	switch action.Type {
	case interactive.ActionPlayLand:
		return mage.PriorityAction{Type: mage.PriorityPlayLand, CardID: action.CardID}
	case interactive.ActionCastSpell:
		return mage.PriorityAction{
			Type:    mage.PriorityCastSpell,
			CardID:  action.CardID,
			Targets: action.Targets,
			XValue:  action.XValue,
		}
	case interactive.ActionActivateAbility:
		return mage.PriorityAction{
			Type:        mage.PriorityActivateAbility,
			PermanentID: action.PermanentID,
			AbilityIdx:  action.AbilityIndex,
			Targets:     action.Targets,
		}
	default:
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
}

// playSingleGame plays one complete game between stratA and stratB.
// Returns 1 if player A wins, 2 if player B wins, 0 if draw.
func playSingleGame(stratA, stratB ai.AIStrategy, deckAEntries, deckBEntries []tui.DeckEntry, maxTurns int) int {
	playerA := ai.NewAIPlayer("PlayerA", stratA)
	playerB := ai.NewAIPlayer("PlayerB", stratB)

	cardsA := tui.BuildDeck(deckAEntries, playerA.PlayerID())
	cardsB := tui.BuildDeck(deckBEntries, playerB.PlayerID())
	for _, c := range cardsA {
		playerA.AddToLibrary(c)
	}
	for _, c := range cardsB {
		playerB.AddToLibrary(c)
	}

	g := mage.NewGame(playerA, playerB)
	tui.DrawOpeningHand(playerA)
	tui.DrawOpeningHand(playerB)
	ai.MulliganAI(playerA)
	ai.MulliganAI(playerB)

	g.SetOnPriority(func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		p := g.PlayerAt(playerIdx)
		aiPlayer, ok := p.(interactive.AutoPlayer)
		if !ok {
			return mage.PriorityAction{Type: mage.PriorityPass}
		}
		isActive := playerIdx == g.ActivePlayerIndex()
		effectiveMainPhase := mainPhase && isActive
		action := aiPlayer.GetPriorityAction(g, g.GetLandsPlayedThisTurn(), effectiveMainPhase)
		return convertActionToPriority(action)
	})

	for g.CurrentTurn() <= maxTurns {
		for _, step := range core.AllSteps() {
			g.RunStepWithPriority(step)
			if g.IsGameOver() {
				winner := g.Winner()
				if winner == playerA.Name() {
					return 1
				} else if winner == playerB.Name() {
					return 2
				}
				return 0
			}
		}
		if g.HasExtraTurns() {
			extraPlayerID, _ := g.PopExtraTurn()
			for i, p := range g.AllPlayers() {
				if p.PlayerID() == extraPlayerID {
					g.SetActivePlayerIndex(i)
					break
				}
			}
		} else {
			g.SetActivePlayerIndex((g.ActivePlayerIndex() + 1) % g.PlayerCount())
		}
		g.SetTurn(g.CurrentTurn() + 1)
	}
	return 0
}

// run1kGames runs 1,000 games pitting newFactory against baselineFactory.
// It runs 500 games where New is PlayerA (plays first) and 500 games where Baseline is PlayerA.
// Decks are mirrored/paired evenly across archetype combinations.
func run1kGames(t *testing.T, label string, newFactory, baselineFactory func(wp ai.WeightedPersonality) ai.AIStrategy) (newWinsCount, baselineWinsCount, drawsCount int64, winRate float64) {
	const totalGames = 1000
	const maxTurns = 50

	archetypes := tui.Archetypes
	numArchetypes := len(archetypes)
	if numArchetypes == 0 {
		t.Fatal("no archetypes available")
	}

	var newWins int64
	var baselineWins int64
	var draws int64

	workers := max(runtime.NumCPU(), 1)

	var wg sync.WaitGroup
	gamesPerWorker := totalGames / workers

	var newWinsByDeck [5]int64
	var baselineWinsByDeck [5]int64

	for w := range workers {
		workerIdx := w
		start := workerIdx * gamesPerWorker
		end := start + gamesPerWorker
		if workerIdx == workers-1 {
			end = totalGames
		}

		wg.Add(1)
		go func(startIdx, endIdx int) {
			defer wg.Done()
			for i := startIdx; i < endIdx; i++ {
				pairIdx := i / 2
				rng := rand.New(rand.NewSource(int64(pairIdx*7919 + 42)))
				deckIdxA := rng.Intn(numArchetypes)
				deckIdxB := rng.Intn(numArchetypes)
				deckA := archetypes[deckIdxA].Entries
				deckB := archetypes[deckIdxB].Entries

				wpA := ai.InferPersonalityFromDeck(aiDeckEntries(deckA))
				wpB := ai.InferPersonalityFromDeck(aiDeckEntries(deckB))

				// Half the games New is PlayerA, half the games New is PlayerB.
				if i%2 == 0 {
					stratA := newFactory(wpA)
					stratB := baselineFactory(wpB)
					res := playSingleGame(stratA, stratB, deckA, deckB, maxTurns)
					switch res {
					case 1:
						atomic.AddInt64(&newWins, 1)
						atomic.AddInt64(&newWinsByDeck[deckIdxA], 1)
					case 2:
						atomic.AddInt64(&baselineWins, 1)
						atomic.AddInt64(&baselineWinsByDeck[deckIdxB], 1)
					default:
						atomic.AddInt64(&draws, 1)
					}
				} else {
					stratA := baselineFactory(wpA)
					stratB := newFactory(wpB)
					res := playSingleGame(stratA, stratB, deckA, deckB, maxTurns)
					switch res {
					case 1:
						atomic.AddInt64(&baselineWins, 1)
						atomic.AddInt64(&baselineWinsByDeck[deckIdxA], 1)
					case 2:
						atomic.AddInt64(&newWins, 1)
						atomic.AddInt64(&newWinsByDeck[deckIdxB], 1)
					default:
						atomic.AddInt64(&draws, 1)
					}
				}
			}
		}(start, end)
	}

	wg.Wait()

	decidedGames := newWins + baselineWins
	winRate = 0.0
	if decidedGames > 0 {
		winRate = float64(newWins) / float64(decidedGames) * 100.0
	}

	fmt.Printf("\n=======================================================\n")
	fmt.Printf("1k Games Simulation Results: %s\n", label)
	fmt.Printf("New AI Wins:      %d (%.1f%% of decided)\n", newWins, winRate)
	fmt.Printf("Baseline AI Wins: %d (%.1f%% of decided)\n", baselineWins, 100.0-winRate)
	for d := range numArchetypes {
		nw := atomic.LoadInt64(&newWinsByDeck[d])
		bw := atomic.LoadInt64(&baselineWinsByDeck[d])
		tot := nw + bw
		wr := 0.0
		if tot > 0 {
			wr = float64(nw) / float64(tot) * 100.0
		}
		fmt.Printf("  %-25s: New %3d vs Base %3d (%.1f%%)\n", archetypes[d].Name, nw, bw, wr)
	}
	fmt.Printf("Draws:            %d\n", draws)
	fmt.Printf("Total Games:      %d\n", totalGames)
	fmt.Printf("=======================================================\n\n")

	return newWins, baselineWins, draws, winRate
}

func aiDeckEntries(entries []tui.DeckEntry) []ai.DeckCard {
	deck := make([]ai.DeckCard, 0, len(entries))
	for _, entry := range entries {
		deck = append(deck, ai.DeckCard{Name: entry.Name, Count: entry.Count})
	}
	return deck
}

func TestSimulation_BaselineVsBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1k simulation in short mode")
	}
	run1kGames(t, "Baseline vs Baseline (Control)", func(wp ai.WeightedPersonality) ai.AIStrategy {
		return newBaselineStrategy(wp)
	}, func(wp ai.WeightedPersonality) ai.AIStrategy {
		return newBaselineStrategy(wp)
	})
}

func TestSimulation_CombatTricksAndTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1k simulation in short mode")
	}
	run1kGames(t, "Combat Pump & Phase Timing vs Baseline", func(wp ai.WeightedPersonality) ai.AIStrategy {
		return New(wp)
	}, func(wp ai.WeightedPersonality) ai.AIStrategy {
		return newBaselineStrategy(wp)
	})
}
