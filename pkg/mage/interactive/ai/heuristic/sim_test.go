package heuristic

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/benprew/mage-go/cards"
	"github.com/benprew/mage-go/internal/scenario"
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

const simulationReferenceTurns = 50
const defaultSimulationMaxTurns = 200
const defaultSimulationGames = 30000

type simulationGameResult struct {
	winner                int
	turnLimited           bool
	unresolvedAtReference bool
}

func prepareSimulationPlayer(player *ai.AIPlayer, entries []tui.DeckEntry, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	var cards []*mage.Card
	for _, entry := range entries {
		for range entry.Count {
			card, err := mage.CreateCard(entry.Name)
			if err != nil {
				panic(err)
			}
			card.SetOwner(player.PlayerID())
			cards = append(cards, card)
		}
	}
	rng.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	player.SetLibrary(cards)
	tui.DrawOpeningHand(player)
	for len(player.Hand()) > 5 && player.ShouldMulligan(len(player.Hand())) {
		handSize := len(player.Hand())
		cards = append(append([]*mage.Card(nil), player.Library()...), player.Hand()...)
		player.SetHand(nil)
		rng.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
		player.SetLibrary(cards)
		for range handSize - 1 {
			player.DrawCard()
		}
	}
}

// playSingleGame keeps the same deck order when a pair swaps AI seats.
func playSingleGame(stratA, stratB ai.AIStrategy, deckAEntries, deckBEntries []tui.DeckEntry, maxTurns int, seedA, seedB int64) simulationGameResult {
	playerA := ai.NewAIPlayer("PlayerA", stratA)
	playerB := ai.NewAIPlayer("PlayerB", stratB)
	g := mage.NewGame(playerA, playerB)
	prepareSimulationPlayer(playerA, deckAEntries, seedA)
	prepareSimulationPlayer(playerB, deckBEntries, seedB)
	g.SetOnPriority(func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		p := g.PlayerAt(playerIdx)
		aiPlayer, ok := p.(interactive.AutoPlayer)
		if !ok {
			return mage.PriorityAction{Type: mage.PriorityPass}
		}
		action := aiPlayer.GetPriorityAction(g, g.GetLandsPlayedThisTurn(), mainPhase && playerIdx == g.ActivePlayerIndex())
		return convertActionToPriority(action)
	})
	unresolvedAtReference := false
	for g.CurrentTurn() <= maxTurns {
		for _, step := range core.AllSteps() {
			g.RunStepWithPriority(step)
			if g.IsGameOver() {
				result := simulationGameResult{unresolvedAtReference: unresolvedAtReference}
				switch g.Winner() {
				case playerA.Name():
					result.winner = 1
				case playerB.Name():
					result.winner = 2
				}
				return result
			}
		}
		if g.CurrentTurn() == simulationReferenceTurns {
			unresolvedAtReference = true
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
	return simulationGameResult{turnLimited: true, unresolvedAtReference: unresolvedAtReference}
}

func getSimulationGameCount() int {
	if s := os.Getenv("SIM_GAMES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return defaultSimulationGames
}

func getSimulationTurnLimit() int {
	if n, err := strconv.Atoi(os.Getenv("SIM_MAX_TURNS")); err == nil && n > 0 {
		return n
	}
	return defaultSimulationMaxTurns
}

func getSimulationSeed() int64 {
	if n, err := strconv.ParseInt(os.Getenv("SIM_SEED"), 10, 64); err == nil {
		return n
	}
	return 42
}

func simulationArchetypes(t *testing.T) []tui.DeckArchetype {
	t.Helper()
	var decks []tui.DeckArchetype
	for _, deck := range tui.Archetypes {
		if deck.Name == "Priestess (W walls)" && os.Getenv("SIM_INCLUDE_PRIESTESS") != "1" {
			continue
		}
		decks = append(decks, deck)
	}
	rogue, err := scenario.ParseDCKDeck("../../../../../rogue_dck/merfolk_shaman.dck")
	if err != nil {
		t.Fatalf("load Merfolk Shaman benchmark deck: %v", err)
	}
	merfolk := tui.DeckArchetype{Name: rogue.Name}
	for _, entry := range rogue.MainCards {
		merfolk.Entries = append(merfolk.Entries, tui.DeckEntry{Name: entry.Name, Count: entry.Count})
	}
	return append(decks, merfolk)
}

type simulationTally struct {
	newWins, baselineWins, draws, turnLimited, unresolvedAtReference int64
	pairs                                                            [3][3]int64
}

func simulationPairedInterval(newWins, oldWins int64, pairs [3][3]int64) (float64, float64, bool) {
	decided := float64(newWins + oldWins)
	if decided == 0 {
		return 0, 0, false
	}
	p := float64(newWins) / decided
	var count, residualSquares float64
	for wins := range 3 {
		for losses := range 3 {
			n := float64(pairs[wins][losses])
			count += n
			residual := float64(wins) - p*float64(wins+losses)
			residualSquares += n * residual * residual
		}
	}
	if count < 30 || residualSquares == 0 {
		return 0, 0, false
	}
	// Keep the two games in each pair together when estimating variance.
	se := math.Sqrt(count * residualSquares / ((count - 1) * decided * decided))
	half := 1.959963984540054 * se
	return 100 * max(0, p-half), 100 * min(1, p+half), true
}

func (s *simulationTally) record(result simulationGameResult, newIsPlayerA bool) {
	if result.unresolvedAtReference {
		s.unresolvedAtReference++
	}
	switch {
	case result.turnLimited:
		s.turnLimited++
	case result.winner == 0:
		s.draws++
	case (result.winner == 1) == newIsPlayerA:
		s.newWins++
	default:
		s.baselineWins++
	}
}

// run1kGames repeats each mirror setup with the AI strategies in opposite seats.
// SIM_GAMES controls the count. SIM_MAX_TURNS and SIM_SEED control the setup.
// SIM_INCLUDE_PRIESTESS=1 adds the slow wall deck.
func run1kGames(t *testing.T, label string, newFactory, baselineFactory func(ai.WeightedPersonality) ai.AIStrategy) (newWinsCount, baselineWinsCount, drawsCount int64, winRate float64) {
	totalGames := getSimulationGameCount()
	if totalGames%2 != 0 {
		t.Fatal("SIM_GAMES must be even for paired mirror matches")
	}
	maxTurns := getSimulationTurnLimit()
	baseSeed := getSimulationSeed()
	archetypes := simulationArchetypes(t)
	if len(archetypes) == 0 {
		t.Fatal("no archetypes available")
	}
	pairs := totalGames / 2
	workers := max(min(runtime.NumCPU(), pairs), 1)
	tallies := make([]simulationTally, len(archetypes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var completed atomic.Int64
	for worker := range workers {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			local := make([]simulationTally, len(archetypes))
			for pair := worker; pair < pairs; pair += workers {
				deckIndex := pair % len(archetypes)
				deck := archetypes[deckIndex].Entries
				weights := ai.InferPersonalityFromDeck(aiDeckEntries(deck))
				seedA := baseSeed + int64(pair)*7919
				seedB := seedA + 1
				beforeWins, beforeLosses := local[deckIndex].newWins, local[deckIndex].baselineWins
				for seat := range 2 {
					var stratA, stratB ai.AIStrategy
					if seat == 0 {
						stratA, stratB = newFactory(weights), baselineFactory(weights)
					} else {
						stratA, stratB = baselineFactory(weights), newFactory(weights)
					}
					result := playSingleGame(stratA, stratB, deck, deck, maxTurns, seedA, seedB)
					local[deckIndex].record(result, seat == 0)
				}
				pairWins := local[deckIndex].newWins - beforeWins
				pairLosses := local[deckIndex].baselineWins - beforeLosses
				local[deckIndex].pairs[pairWins][pairLosses]++
				if n := completed.Add(2); totalGames >= 1000 && n%1000 == 0 {
					fmt.Printf("Completed %d/%d mirror games\n", n, totalGames)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			for i, s := range local {
				tallies[i].newWins += s.newWins
				tallies[i].baselineWins += s.baselineWins
				tallies[i].draws += s.draws
				tallies[i].turnLimited += s.turnLimited
				tallies[i].unresolvedAtReference += s.unresolvedAtReference
				for wins := range 3 {
					for losses := range 3 {
						tallies[i].pairs[wins][losses] += s.pairs[wins][losses]
					}
				}
			}
		}(worker)
	}
	wg.Wait()
	var actualDraws, turnLimited, unresolvedAtReference int64
	var pairCounts [3][3]int64
	for _, s := range tallies {
		newWinsCount += s.newWins
		baselineWinsCount += s.baselineWins
		actualDraws += s.draws
		turnLimited += s.turnLimited
		unresolvedAtReference += s.unresolvedAtReference
		for wins := range 3 {
			for losses := range 3 {
				pairCounts[wins][losses] += s.pairs[wins][losses]
			}
		}
	}
	drawsCount = actualDraws + turnLimited
	if got := newWinsCount + baselineWinsCount + drawsCount; got != int64(totalGames) {
		t.Fatalf("expected %d games, got %d", totalGames, got)
	}
	baselineWinRate := 0.0
	if decided := newWinsCount + baselineWinsCount; decided > 0 {
		winRate = 100 * float64(newWinsCount) / float64(decided)
		baselineWinRate = 100 * float64(baselineWinsCount) / float64(decided)
	}
	if totalGames >= 10 {
		fmt.Printf("\n=======================================================\n")
		fmt.Printf("Simulation Results (%d games): %s\n", totalGames, label)
		fmt.Printf("Matchups: paired mirrors; maximum %d individual turns; seed %d\n", maxTurns, baseSeed)
		fmt.Printf("New AI Wins:      %d (%.1f%% of decided)\n", newWinsCount, winRate)
		fmt.Printf("Baseline AI Wins: %d (%.1f%% of decided)\n", baselineWinsCount, baselineWinRate)
		if lower, upper, ok := simulationPairedInterval(newWinsCount, baselineWinsCount, pairCounts); ok {
			fmt.Printf("95%% paired confidence interval: %.4f%% to %.4f%%\n", lower, upper)
		}
		for wins := range 3 {
			for losses := range 3 {
				if n := pairCounts[wins][losses]; n > 0 {
					fmt.Printf("Pair outcome: New wins %d; Base wins %d; Pairs %d\n", wins, losses, n)
				}
			}
		}
		for i, s := range tallies {
			rate := 0.0
			if decided := s.newWins + s.baselineWins; decided > 0 {
				rate = 100 * float64(s.newWins) / float64(decided)
			}
			games := s.newWins + s.baselineWins + s.draws + s.turnLimited
			fmt.Printf("  %-25s: New %d vs Base %d (%.1f%%); Draws %d; Turn limit %d; Unfinished at %d %d; Games %d\n", archetypes[i].Name, s.newWins, s.baselineWins, rate, s.draws, s.turnLimited, simulationReferenceTurns, s.unresolvedAtReference, games)
		}
		fmt.Printf("Draws:            %d (%.2f%%; actual %d; turn limit %d)\n", drawsCount, 100*float64(drawsCount)/float64(totalGames), actualDraws, turnLimited)
		fmt.Printf("Unfinished at %d turns: %d (%.2f%%)\n", simulationReferenceTurns, unresolvedAtReference, 100*float64(unresolvedAtReference)/float64(totalGames))
		fmt.Printf("Total Games:      %d\n", totalGames)
		fmt.Printf("=======================================================\n\n")
	}
	return
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
		t.Skip("skipping full AI comparison in short mode")
	}
	run1kGames(t, "Baseline vs Baseline (Control)", func(wp ai.WeightedPersonality) ai.AIStrategy {
		return newBaselineStrategy(wp)
	}, func(wp ai.WeightedPersonality) ai.AIStrategy {
		return newBaselineStrategy(wp)
	})
}

func TestSimulation_CombatTricksAndTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full AI comparison in short mode")
	}
	run1kGames(t, "Combat Pump & Phase Timing vs Baseline", func(wp ai.WeightedPersonality) ai.AIStrategy {
		return New(wp)
	}, func(wp ai.WeightedPersonality) ai.AIStrategy {
		return newBaselineStrategy(wp)
	})
}
