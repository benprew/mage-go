// Command scenariotest runs batches of AI vs AI games using Shandalar rogue
// decks and emits structured JSONL results for analysis.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/benprew/mage-go/internal/scenario"
	"github.com/benprew/mage-go/internal/tui"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"

	_ "github.com/benprew/mage-go/cards"
)

func main() {
	roguesDir := flag.String("rogues-dir", "../s30/assets/configs/rogues", "path to Shandalar rogues TOML directory")
	numGames := flag.Int("games", 100, "number of games to run")
	maxTurns := flag.Int("max-turns", 50, "per-game turn limit")
	output := flag.String("output", "", "output JSONL file (default: stdout)")
	seed := flag.Int64("seed", 0, "master RNG seed (0 = time-based)")
	aiPers := flag.String("ai", "auto", "AI personality (auto, aggro, control, midrange, tempo, burn, adaptive)")
	aiPersA := flag.String("ai-a", "", "AI personality for player A (default: -ai)")
	aiPersB := flag.String("ai-b", "", "AI personality for player B (default: -ai)")
	modeA := flag.String("mode-a", "heuristic", "AI mode for player A (heuristic, adaptive)")
	modeB := flag.String("mode-b", "heuristic", "AI mode for player B (heuristic, adaptive)")
	timeout := flag.Duration("timeout", 30*time.Second, "per-game wall clock timeout")
	minCards := flag.Int("min-cards", 25, "minimum playable cards for a deck to be eligible")
	flag.Parse()
	if *aiPersA == "" {
		*aiPersA = *aiPers
	}
	if *aiPersB == "" {
		*aiPersB = *aiPers
	}

	decks, err := scenario.LoadAllRogueDecks(*roguesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading decks: %v\n", err)
		os.Exit(1)
	}

	type filteredDeck struct {
		deck    *scenario.RogueDeck
		entries []tui.DeckEntry
		skipped []string
	}

	var eligible []filteredDeck
	for _, d := range decks {
		entries, skipped := filterDeck(d)
		total := 0
		for _, e := range entries {
			total += e.Count
		}
		if total < *minCards {
			fmt.Fprintf(os.Stderr, "skipping %s: only %d playable cards (need %d)\n", d.Name, total, *minCards)
			continue
		}
		eligible = append(eligible, filteredDeck{deck: d, entries: entries, skipped: skipped})
	}

	if len(eligible) == 0 {
		fmt.Fprintf(os.Stderr, "no eligible decks found\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "loaded %d eligible decks (from %d total)\n", len(eligible), len(decks))

	var out *os.File
	if *output != "" {
		out, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating output: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = out.Close() }()
	} else {
		out = os.Stdout
	}
	enc := json.NewEncoder(out)

	masterSeed := *seed
	if masterSeed == 0 {
		masterSeed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(masterSeed))
	fmt.Fprintf(os.Stderr, "master seed: %d\n", masterSeed)

	for i := 0; i < *numGames; i++ {
		dA := eligible[rng.Intn(len(eligible))]
		dB := eligible[rng.Intn(len(eligible))]
		gameSeed := rng.Int63()

		result := runOneGame(gameConfig{
			gameID:   i + 1,
			seed:     gameSeed,
			deckA:    dA,
			deckB:    dB,
			maxTurns: *maxTurns,
			aiPersA:  *aiPersA,
			aiPersB:  *aiPersB,
			modeA:    *modeA,
			modeB:    *modeB,
			timeout:  *timeout,
		})

		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "error writing result: %v\n", err)
		}
		fmt.Fprintf(os.Stderr, "game %d/%d: %s vs %s → %s (%s, turn %d)\n",
			i+1, *numGames, dA.deck.Name, dB.deck.Name,
			result.Winner, result.WinReason, result.FinalTurn)
	}
}

type gameConfig struct {
	gameID int
	seed   int64
	deckA  struct {
		deck    *scenario.RogueDeck
		entries []tui.DeckEntry
		skipped []string
	}
	deckB struct {
		deck    *scenario.RogueDeck
		entries []tui.DeckEntry
		skipped []string
	}
	maxTurns int
	aiPersA  string
	aiPersB  string
	modeA    string
	modeB    string
	timeout  time.Duration
}

func runOneGame(cfg gameConfig) (result scenario.GameResult) {
	result.GameID = cfg.gameID
	result.Seed = cfg.seed
	result.DeckA = cfg.deckA.deck.Name
	result.DeckB = cfg.deckB.deck.Name
	result.DeckAFile = cfg.deckA.deck.SourceFile
	result.DeckBFile = cfg.deckB.deck.SourceFile
	wpA := resolvePersonality(cfg.aiPersA, cfg.deckA.entries)
	wpB := resolvePersonality(cfg.aiPersB, cfg.deckB.entries)
	result.AIPersonalityA = personalityLabel(cfg.aiPersA, wpA) + "/" + cfg.modeA
	result.AIPersonalityB = personalityLabel(cfg.aiPersB, wpB) + "/" + cfg.modeB
	result.CardsSkippedA = cfg.deckA.skipped
	result.CardsSkippedB = cfg.deckB.skipped

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start).Round(time.Millisecond).String()
	}()

	defer func() {
		if r := recover(); r != nil {
			result.PanicMsg = fmt.Sprintf("%v", r)
			result.PanicStack = string(debug.Stack())
			result.WinReason = "panic"
		}
	}()

	gameRng := rand.New(rand.NewSource(cfg.seed))

	playerA := createAI("Alice", cfg.modeA, wpA)
	playerB := createAI("Bob", cfg.modeB, wpB)

	cardsA := buildDeckFromEntries(cfg.deckA.entries, playerA.PlayerID(), gameRng)
	cardsB := buildDeckFromEntries(cfg.deckB.entries, playerB.PlayerID(), gameRng)
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

	actionCount := 0
	var anomalies []string

	g.SetOnPriority(func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		p := g.PlayerAt(playerIdx)
		aiPlayer := p.(interactive.AutoPlayer)
		isActive := playerIdx == g.ActivePlayerIndex()
		effectiveMainPhase := mainPhase && isActive
		action := aiPlayer.GetPriorityAction(g, g.GetLandsPlayedThisTurn(), effectiveMainPhase)
		return convertAction(action)
	})

	g.SetAfterPriorityAction(func(g *mage.Game, playerIdx int, action mage.PriorityAction) {
		if action.Type != mage.PriorityPass {
			actionCount++
		}
	})

	done := make(chan struct{})
	timedOut := false

	go func() {
		select {
		case <-done:
		case <-time.After(cfg.timeout):
			timedOut = true
			// We can't cleanly interrupt the game loop, but we set the flag
			// so the turn loop checks it.
		}
	}()

	for g.CurrentTurn() <= cfg.maxTurns && !timedOut {
		for _, step := range core.AllSteps() {
			g.RunStepWithPriority(step)

			// Anomaly detection after each step.
			for idx := 0; idx < g.PlayerCount(); idx++ {
				p := g.PlayerAt(idx)
				if p.Life() <= 0 && !g.IsGameOver() {
					anomalies = append(anomalies, fmt.Sprintf("negative_life_alive:%s:%d:turn%d",
						p.Name(), p.Life(), g.CurrentTurn()))
				}
			}

			if g.IsGameOver() {
				break
			}
			if timedOut {
				break
			}
		}

		if g.IsGameOver() || timedOut {
			break
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

	close(done)

	result.FinalTurn = g.CurrentTurn()
	result.TotalActions = actionCount
	result.Anomalies = anomalies
	result.FinalLifeA = g.PlayerAt(0).Life()
	result.FinalLifeB = g.PlayerAt(1).Life()

	if timedOut {
		result.WinReason = "timeout"
	} else if g.IsGameOver() {
		result.Winner = g.Winner()
		if g.Winner() != "" {
			result.WinReason = "life"
			for idx := 0; idx < g.PlayerCount(); idx++ {
				p := g.PlayerAt(idx)
				if p.Name() != g.Winner() && len(p.Library()) == 0 {
					result.WinReason = "library"
				}
			}
		}
	} else {
		result.WinReason = "max_turns"
	}

	return result
}

// filterDeck checks which cards in a rogue deck are registered and returns
// playable entries. Unregistered non-land cards are replaced with the deck's
// primary basic land to maintain deck size.
func filterDeck(d *scenario.RogueDeck) (entries []tui.DeckEntry, skipped []string) {
	basicCounts := map[string]int{}
	for _, e := range d.MainCards {
		switch e.Name {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			basicCounts[e.Name] += e.Count
		}
	}

	primaryBasic := "Plains"
	maxCount := 0
	for name, count := range basicCounts {
		if count > maxCount {
			maxCount = count
			primaryBasic = name
		}
	}

	backfill := 0
	for _, e := range d.MainCards {
		if mage.CardRegistered(e.Name) {
			entries = append(entries, tui.DeckEntry{Name: e.Name, Count: e.Count})
		} else {
			skipped = append(skipped, e.Name)
			backfill += e.Count
		}
	}

	if backfill > 0 {
		found := false
		for i, e := range entries {
			if e.Name == primaryBasic {
				entries[i].Count += backfill
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, tui.DeckEntry{Name: primaryBasic, Count: backfill})
		}
	}

	return entries, skipped
}

func buildDeckFromEntries(entries []tui.DeckEntry, ownerID [16]byte, rng *rand.Rand) []mage.Card {
	var deck []mage.Card
	for _, entry := range entries {
		for i := 0; i < entry.Count; i++ {
			card, err := mage.CreateCard(entry.Name)
			if err != nil {
				continue
			}
			card.SetOwner(ownerID)
			deck = append(deck, card)
		}
	}
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

func createAI(name, mode string, wp ai.WeightedPersonality) *ai.AIPlayer {
	if strings.EqualFold(mode, "adaptive") {
		return ai.NewAIPlayer(name, heuristic.NewAdaptive())
	}
	return ai.NewAIPlayer(name, heuristic.New(wp))
}

func parsePersonality(s string) ai.WeightedPersonality {
	switch strings.ToLower(s) {
	case "aggro":
		return ai.AggroWeighted
	case "control":
		return ai.ControlWeighted
	case "midrange":
		return ai.MidrangeWeighted
	case "tempo":
		return ai.TempoWeighted
	case "burn":
		return ai.BurnWeighted
	default:
		return ai.MidrangeWeighted
	}
}

func resolvePersonality(s string, entries []tui.DeckEntry) ai.WeightedPersonality {
	if strings.EqualFold(s, "auto") || strings.EqualFold(s, "deck") {
		return ai.InferPersonalityFromDeck(aiDeckEntries(entries))
	}
	return parsePersonality(s)
}

func personalityLabel(requested string, wp ai.WeightedPersonality) string {
	if strings.EqualFold(requested, "auto") || strings.EqualFold(requested, "deck") {
		return "auto:" + strings.ToLower(wp.Name)
	}
	return requested
}

func aiDeckEntries(entries []tui.DeckEntry) []ai.DeckCard {
	deck := make([]ai.DeckCard, 0, len(entries))
	for _, entry := range entries {
		deck = append(deck, ai.DeckCard{Name: entry.Name, Count: entry.Count})
	}
	return deck
}

func convertAction(action interactive.PriorityAction) mage.PriorityAction {
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
